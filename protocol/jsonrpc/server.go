/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package jsonrpc

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/opentracing/opentracing-go"
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// A value sent as a placeholder for the server's response value when the server
// receives an invalid request. It is never decoded by the client since the Response
// contains an error when it is used.
var invalidRequest = struct{}{}

const (
	// DefaultMaxSleepTime max sleep interval in accept
	DefaultMaxSleepTime = 1 * time.Second
	// DefaultHTTPRspBufferSize ...
	DefaultHTTPRspBufferSize = 1024
	// PathPrefix ...
	PathPrefix = byte('/')
	// Max HTTP header size in Mib
	MaxHeaderSize = 8 * 1024 * 1024
	// ContentTypeHeader is the HTTP Content-Type header name.
	ContentTypeHeader = "Content-Type"
	// maxRequestWindowPerConnection bounds requests whose responses have not been written yet.
	maxRequestWindowPerConnection = 64
)

// Server is JSON RPC server wrapper
type Server struct {
	done chan struct{}
	once sync.Once

	sync.RWMutex
	wg sync.WaitGroup
}

// NewServer creates new JSON RPC server.
func NewServer() *Server {
	return &Server{
		done: make(chan struct{}),
	}
}

func (s *Server) handlePkg(conn net.Conn) {
	connectionCtx, connectionCancel := context.WithCancel(context.Background())
	responses := make(chan orderedResponse)
	requestWindow := make(chan struct{}, maxRequestWindowPerConnection)
	responseWriterDone := make(chan struct{})
	go func() {
		defer close(responseWriterDone)
		writeResponsesInOrder(connectionCtx, connectionCancel, conn, responses, requestWindow)
	}()
	var requestWG sync.WaitGroup
	defer func() {
		if r := recover(); r != nil {
			logger.Warnf("[Jsonrpc][Server] connection panic, local=%v, remote=%v, err=%v, debug stack=%s",
				conn.LocalAddr(), conn.RemoteAddr(), r, string(debug.Stack()))
		}

		conn.Close()
		requestWG.Wait()
		<-responseWriterDone
	}()
	// Register this after the cleanup defer so LIFO ordering cancels request contexts before Wait.
	defer connectionCancel()

	limitedReader := &io.LimitedReader{R: conn}
	bufReader := bufio.NewReader(limitedReader)
	var sequence uint64
	for {
		select {
		case requestWindow <- struct{}{}:
		case <-connectionCtx.Done():
			return
		}

		limitedReader.N = int64(MaxHeaderSize - bufReader.Buffered())
		if _, err := bufReader.Peek(1); errors.Is(err, io.EOF) {
			<-requestWindow
			return
		}
		r, err := http.ReadRequest(bufReader)
		if err != nil {
			<-requestWindow
			logger.Warnf("[Jsonrpc][Server] read request failed, err=%v", err)
			return
		}

		reqBody, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			<-requestWindow
			return
		}

		reqHeader := make(map[string]string)
		for k := range r.Header {
			reqHeader[k] = r.Header.Get(k)
		}
		reqHeader["Path"] = r.URL.Path[1:] // to get service name
		if r.URL.Path[0] != PathPrefix {
			reqHeader["Path"] = r.URL.Path
		}
		reqHeader["HttpMethod"] = r.Method

		contentType := reqHeader[ContentTypeHeader]
		mediaType, _, parseErr := mime.ParseMediaType(contentType)
		unsupportedContentType := parseErr != nil || (mediaType != "application/json" && mediaType != "application/json-rpc")

		requestCtx, requestCancel := context.WithCancel(connectionCtx)
		r = r.WithContext(requestCtx)
		ctx := contextFromRequest(r)
		var timeoutCancel context.CancelFunc

		if len(reqHeader["Timeout"]) > 0 {
			timeout, err := time.ParseDuration(reqHeader["Timeout"])
			if err == nil {
				ctx, timeoutCancel = context.WithTimeout(ctx, timeout)
			}
			delete(reqHeader, "Timeout")
		}

		requestSequence := sequence
		sequence++
		requestWG.Add(1)
		go func(ctx context.Context, requestCancel, timeoutCancel context.CancelFunc, header map[string]string, body []byte,
			responseHeader http.Header, contentType string, unsupportedContentType bool, requestSequence uint64) {
			defer requestWG.Done()
			defer requestCancel()
			if timeoutCancel != nil {
				defer timeoutCancel()
			}

			response := buildOrderedResponse(ctx, header, body, responseHeader, contentType,
				unsupportedContentType, requestSequence)
			select {
			case responses <- response:
			case <-connectionCtx.Done():
			}
		}(ctx, requestCancel, timeoutCancel, reqHeader, reqBody, r.Header, contentType, unsupportedContentType, requestSequence)
	}
}

type orderedResponse struct {
	sequence        uint64
	data            []byte
	closeConnection bool
}

func buildOrderedResponse(ctx context.Context, header map[string]string, body []byte, responseHeader http.Header,
	contentType string, unsupportedContentType bool, sequence uint64) orderedResponse {
	responseBuffer := bytes.NewBuffer(nil)
	if unsupportedContentType {
		errMsg := "unsupported content type: " + contentType
		if err := writeHTTPErrorResponse(responseBuffer, responseHeader, []byte(errMsg)); err != nil {
			logger.Warnf("[Jsonrpc][Server] write error response failed, header=%v, err_msg=%v, write_err=%v",
				responseHeader, errMsg, err)
		}
		return orderedResponse{sequence: sequence, data: responseBuffer.Bytes(), closeConnection: true}
	}

	err := serveRequest(ctx, header, body, responseBuffer)
	if err == nil {
		return orderedResponse{sequence: sequence, data: responseBuffer.Bytes()}
	}
	if writeErr := writeHTTPErrorResponse(responseBuffer, responseHeader, []byte(perrors.WithStack(err).Error())); writeErr != nil {
		logger.Warnf("[Jsonrpc][Server] write error response failed, header=%v, err=%v, write_err=%v",
			responseHeader, perrors.WithStack(err), writeErr)
	}
	logger.Infof("[Jsonrpc][Server] unexpected error serving request, closing socket, err=%v", err)
	return orderedResponse{sequence: sequence, data: responseBuffer.Bytes(), closeConnection: true}
}

func writeHTTPErrorResponse(writer io.Writer, header http.Header, body []byte) error {
	rsp := &http.Response{
		Header:        header.Clone(),
		StatusCode:    500,
		ProtoMajor:    1,
		ProtoMinor:    1,
		ContentLength: int64(len(body)),
		Body:          io.NopCloser(bytes.NewReader(body)),
	}
	rsp.Header.Del(ContentTypeHeader)
	rsp.Header.Del("Content-Length")
	rsp.Header.Del("Timeout")

	rspBuf := bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize))
	rspBuf.Reset()
	if err := rsp.Write(rspBuf); err != nil {
		return perrors.WithStack(err)
	}
	_, err := rspBuf.WriteTo(writer)
	return perrors.WithStack(err)
}

func writeResponsesInOrder(ctx context.Context, cancel context.CancelFunc, conn net.Conn, responses <-chan orderedResponse,
	requestWindow <-chan struct{}) {
	nextSequence := uint64(0)
	pending := make(map[uint64]orderedResponse)
	for {
		select {
		case <-ctx.Done():
			return
		case response := <-responses:
			pending[response.sequence] = response
		}

		for {
			response, ok := pending[nextSequence]
			if !ok {
				break
			}
			delete(pending, nextSequence)
			if _, err := bytes.NewReader(response.data).WriteTo(conn); err != nil {
				logger.Warnf("[Jsonrpc][Server] write response failed, sequence=%d, err=%v", nextSequence, err)
				cancel()
				conn.Close()
				return
			}
			if response.closeConnection {
				cancel()
				conn.Close()
				return
			}
			nextSequence++
			<-requestWindow
		}
	}
}

func contextFromRequest(r *http.Request) context.Context {
	ctx := r.Context()
	spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.HTTPHeaders,
		opentracing.HTTPHeadersCarrier(r.Header))
	if err == nil {
		ctx = context.WithValue(ctx, constant.TracingRemoteSpanCtx, spanCtx)
	}
	return ctx
}

func accept(listener net.Listener, fn func(net.Conn)) error {
	var (
		ok       bool
		ne       net.Error
		tmpDelay time.Duration
	)

	for {
		c, err := listener.Accept()
		if err != nil {
			if ne, ok = err.(net.Error); ok && ne.Temporary() {
				if tmpDelay != 0 {
					tmpDelay <<= 1
				} else {
					tmpDelay = 5 * time.Millisecond
				}
				if tmpDelay > DefaultMaxSleepTime {
					tmpDelay = DefaultMaxSleepTime
				}
				logger.Infof("[Jsonrpc][Server] http accept error, retrying, err=%v, retry_in=%v", err, tmpDelay)
				time.Sleep(tmpDelay)
				continue
			}
			return perrors.WithStack(err)
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					logger.Errorf("[Jsonrpc][Server] http panic serving, remote=%v, err=%v\n%s", c.RemoteAddr(), r, buf)
					c.Close()
				}
			}()

			fn(c)
		}()
	}
}

// Start JSON RPC server then ready for accept request.
func (s *Server) Start(url *common.URL) {
	listener, err := net.Listen("tcp", url.Location)
	if err != nil {
		logger.Errorf("[Jsonrpc][Server] jsonrpc server start failed, path=%s, err=%v", url.Path, err)
		return
	}
	logger.Infof("[Jsonrpc][Server] rpc server start to listen on %s", listener.Addr())

	s.wg.Go(func() {
		if err := accept(listener, func(conn net.Conn) { s.handlePkg(conn) }); err != nil {
			logger.Errorf("[Jsonrpc][Server] accept failed, err=%v", err)
		}
	})

	s.wg.Go(func() { // Server done goroutine
		var err error
		<-s.done               // step1: block to wait for done channel(wait Server.Stop step2)
		err = listener.Close() // step2: and then close listener
		if err != nil {
			logger.Warnf("[Jsonrpc][Server] listener close failed, addr=%s, err=%v", listener.Addr(), err)
		}
	})
}

// Stop JSON RPC server, just can be call once.
func (s *Server) Stop() {
	s.once.Do(func() {
		close(s.done)
		s.wg.Wait()
	})
}

func serveRequest(ctx context.Context, header map[string]string, body []byte, writer io.Writer) error {
	sendErrorResp := func(header map[string]string, body []byte) error {
		rsp := &http.Response{
			Header:        make(http.Header),
			StatusCode:    500,
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: int64(len(body)),
			Body:          io.NopCloser(bytes.NewReader(body)),
		}
		rsp.Header.Del(ContentTypeHeader)
		rsp.Header.Del("Content-Length")
		rsp.Header.Del("Timeout")
		for k, v := range header {
			rsp.Header.Set(k, v)
		}

		rspBuf := bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize))
		rspBuf.Reset()
		err := rsp.Write(rspBuf)
		if err != nil {
			return perrors.WithStack(err)
		}
		_, err = rspBuf.WriteTo(writer)
		return perrors.WithStack(err)
	}

	sendResp := func(header map[string]string, body []byte) error {
		rsp := &http.Response{
			Header:        make(http.Header),
			StatusCode:    200,
			ProtoMajor:    1,
			ProtoMinor:    1,
			ContentLength: int64(len(body)),
			Body:          io.NopCloser(bytes.NewReader(body)),
		}
		rsp.Header.Del(ContentTypeHeader)
		rsp.Header.Del("Content-Length")
		rsp.Header.Del("Timeout")
		for k, v := range header {
			rsp.Header.Set(k, v)
		}

		rspBuf := bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize))
		rspBuf.Reset()
		err := rsp.Write(rspBuf)
		if err != nil {
			return perrors.WithStack(err)
		}
		_, err = rspBuf.WriteTo(writer)
		return perrors.WithStack(err)
	}

	// read request header
	codec := newServerCodec()
	err := codec.ReadHeader(header, body)
	if err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return perrors.WithStack(err)
		}
		return errors.New("server cannot decode request: " + err.Error())
	}

	path := header["Path"]
	methodName := codec.req.Method
	if len(path) == 0 || len(methodName) == 0 {
		return errors.New("service/method request ill-formed: " + path + "/" + methodName)
	}

	// read body
	var args []any
	if err = codec.ReadBody(&args); err != nil {
		return perrors.WithStack(err)
	}
	logger.Debugf("[Jsonrpc][Server] args=%v", args)

	// exporter invoke
	exporter, ok := jsonrpcProtocol.ExporterMap().Load(path)
	if !ok {
		return fmt.Errorf("service not found: %s", path)
	}
	invoker := exporter.(*JsonrpcExporter).GetInvoker()
	if invoker != nil {
		rpcInvocation := invocation.NewRPCInvocation(methodName, args, map[string]any{
			constant.PathKey:    path,
			constant.VersionKey: codec.req.Version,
		})
		rpcInvocation.SetContext(ctx)
		result := invoker.Invoke(ctx, rpcInvocation)
		if err := result.Error(); err != nil {
			rspStream, codecErr := codec.Write(err.Error(), invalidRequest)
			if codecErr != nil {
				return perrors.WithStack(codecErr)
			}
			if errRsp := sendErrorResp(header, rspStream); errRsp != nil {
				logger.Warnf("[Jsonrpc][Server] sendErrorResp failed, header=%v, err=%v, send_err=%v",
					header, err, errRsp)
			}
		} else {
			res := result.Result()
			rspStream, err := codec.Write("", res)
			if err != nil {
				return perrors.WithStack(err)
			}
			if errRsp := sendResp(header, rspStream); errRsp != nil {
				logger.Warnf("[Jsonrpc][Server] sendResp failed, header=%v, err=%v, send_err=%v",
					header, err, errRsp)
			}
		}
	}

	return nil
}
