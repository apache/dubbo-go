package jsonrpc

import (
	"bufio"
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"runtime"
	"runtime/debug"
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	"github.com/dubbo/go-for-apache-dubbo/protocol/invocation"
)

var (
	// A value sent as a placeholder for the server's response value when the server
	// receives an invalid request. It is never decoded by the client since the Response
	// contains an error when it is used.
	invalidRequest = struct{}{}
)

const (
	DefaultMaxSleepTime      = 1 * time.Second // accept中间最大sleep interval
	DefaultHTTPRspBufferSize = 1024
	PathPrefix               = byte('/')
)

type Server struct {
	exporter protocol.Exporter
	done     chan struct{}
	once     sync.Once

	sync.RWMutex
	wg      sync.WaitGroup
	timeout time.Duration
}

func NewServer(exporter protocol.Exporter) *Server {
	return &Server{
		exporter: exporter,
		done:     make(chan struct{}),
	}
}

func (s *Server) handlePkg(conn net.Conn) {
	defer func() {
		if r := recover(); r != nil {
			log.Warn("connection{local:%v, remote:%v} panic error:%#v, debug stack:%s",
				conn.LocalAddr(), conn.RemoteAddr(), r, string(debug.Stack()))
		}

		conn.Close()
	}()

	setTimeout := func(conn net.Conn, timeout time.Duration) {
		t := time.Time{}
		if timeout > time.Duration(0) {
			t = time.Now().Add(timeout)
		}

		conn.SetDeadline(t)
	}

	sendErrorResp := func(header http.Header, body []byte) error {
		rsp := &http.Response{
			Header:        header,
			StatusCode:    500,
			ContentLength: int64(len(body)),
			Body:          ioutil.NopCloser(bytes.NewReader(body)),
		}
		rsp.Header.Del("Content-Type")
		rsp.Header.Del("Content-Length")
		rsp.Header.Del("Timeout")

		rspBuf := bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize))
		rspBuf.Reset()
		err := rsp.Write(rspBuf)
		if err != nil {
			return jerrors.Trace(err)
		}
		_, err = rspBuf.WriteTo(conn)
		return jerrors.Trace(err)
	}

	for {
		bufReader := bufio.NewReader(conn)
		r, err := http.ReadRequest(bufReader)
		if err != nil {
			log.Warn("[ReadRequest] error: %v", err)
			return
		}

		reqBody, err := ioutil.ReadAll(r.Body)
		if err != nil {
			return
		}
		r.Body.Close()

		reqHeader := make(map[string]string)
		for k := range r.Header {
			reqHeader[k] = r.Header.Get(k)
		}
		reqHeader["Path"] = r.URL.Path[1:] // to get service name
		if r.URL.Path[0] != PathPrefix {
			reqHeader["Path"] = r.URL.Path
		}
		reqHeader["HttpMethod"] = r.Method

		httpTimeout := s.timeout
		contentType := reqHeader["Content-Type"]
		if contentType != "application/json" && contentType != "application/json-rpc" {
			setTimeout(conn, httpTimeout)
			r.Header.Set("Content-Type", "text/plain")
			if errRsp := sendErrorResp(r.Header, []byte(jerrors.ErrorStack(err))); errRsp != nil {
				log.Warn("sendErrorResp(header:%#v, error:%s) = error:%s",
					r.Header, jerrors.ErrorStack(err), errRsp)
			}
			return
		}

		ctx := context.Background()
		if len(reqHeader["Timeout"]) > 0 {
			timeout, err := time.ParseDuration(reqHeader["Timeout"])
			if err == nil {
				httpTimeout = timeout
				ctx, _ = context.WithTimeout(ctx, httpTimeout)
			}
			delete(reqHeader, "Timeout")
		}
		setTimeout(conn, httpTimeout)

		if err := serveRequest(ctx, reqHeader, reqBody, conn, s.exporter); err != nil {
			if errRsp := sendErrorResp(r.Header, []byte(jerrors.ErrorStack(err))); errRsp != nil {
				log.Warn("sendErrorResp(header:%#v, error:%s) = error:%s",
					r.Header, jerrors.ErrorStack(err), errRsp)
			}

			log.Info("Unexpected error serving request, closing socket: %v", err)
			return
		}
	}
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
				log.Info("http: Accept error: %v; retrying in %v\n", err, tmpDelay)
				time.Sleep(tmpDelay)
				continue
			}
			return jerrors.Trace(err)
		}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					const size = 64 << 10
					buf := make([]byte, size)
					buf = buf[:runtime.Stack(buf, false)]
					log.Error("http: panic serving %v: %v\n%s", c.RemoteAddr(), r, buf)
					c.Close()
				}
			}()

			fn(c)
		}()
	}
}

func (s *Server) Start(url common.URL) {
	listener, err := net.Listen("tcp", url.Location)
	if err != nil {
		log.Error("jsonrpc server [%s] start failed: %v", url.Path, err)
		return
	}
	log.Info("rpc server start to listen on %s", listener.Addr())

	s.wg.Add(1)
	go func() {
		accept(listener, func(conn net.Conn) { s.handlePkg(conn) })
		s.wg.Done()
	}()

	s.wg.Add(1)
	go func() { // Server done goroutine
		var err error
		<-s.done               // step1: block to wait for done channel(wait Server.Stop step2)
		err = listener.Close() // step2: and then close listener
		if err != nil {
			log.Warn("listener{addr:%s}.Close() = error{%#v}", listener.Addr(), err)
		}
		s.wg.Done()
	}()
}

func (s *Server) Stop() {
	s.once.Do(func() {
		close(s.done)
		s.wg.Wait()
	})
}

func serveRequest(ctx context.Context,
	header map[string]string, body []byte, conn net.Conn, exporter protocol.Exporter) error {

	sendErrorResp := func(header map[string]string, body []byte) error {
		rsp := &http.Response{
			Header:        make(http.Header),
			StatusCode:    500,
			ContentLength: int64(len(body)),
			Body:          ioutil.NopCloser(bytes.NewReader(body)),
		}
		rsp.Header.Del("Content-Type")
		rsp.Header.Del("Content-Length")
		rsp.Header.Del("Timeout")
		for k, v := range header {
			rsp.Header.Set(k, v)
		}

		rspBuf := bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize))
		rspBuf.Reset()
		err := rsp.Write(rspBuf)
		if err != nil {
			return jerrors.Trace(err)
		}
		_, err = rspBuf.WriteTo(conn)
		return jerrors.Trace(err)
	}

	sendResp := func(header map[string]string, body []byte) error {
		rsp := &http.Response{
			Header:        make(http.Header),
			StatusCode:    200,
			ContentLength: int64(len(body)),
			Body:          ioutil.NopCloser(bytes.NewReader(body)),
		}
		rsp.Header.Del("Content-Type")
		rsp.Header.Del("Content-Length")
		rsp.Header.Del("Timeout")
		for k, v := range header {
			rsp.Header.Set(k, v)
		}

		rspBuf := bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize))
		rspBuf.Reset()
		err := rsp.Write(rspBuf)
		if err != nil {
			return jerrors.Trace(err)
		}
		_, err = rspBuf.WriteTo(conn)
		return jerrors.Trace(err)
	}

	// read request header
	codec := newServerCodec()
	err := codec.ReadHeader(header, body)
	if err != nil {
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			return jerrors.Trace(err)
		}

		return jerrors.New("server cannot decode request: " + err.Error())
	}
	serviceName := header["Path"]
	methodName := codec.req.Method
	if len(serviceName) == 0 || len(methodName) == 0 {
		codec.ReadBody(nil)
		return jerrors.New("service/method request ill-formed: " + serviceName + "/" + methodName)
	}

	// read body
	var args interface{}
	if err = codec.ReadBody(&args); err != nil {
		return jerrors.Trace(err)
	}
	log.Debug("args: %v", args)

	// exporter invoke
	invoker := exporter.GetInvoker()
	if invoker != nil {
		result := invoker.Invoke(invocation.NewRPCInvocationForProvider(methodName, args.([]interface{}), map[string]string{
			//attachments[constant.PATH_KEY] = url.Path
			//attachments[constant.GROUP_KEY] = url.GetParam(constant.GROUP_KEY, "")
			//attachments[constant.INTERFACE_KEY] = url.GetParam(constant.INTERFACE_KEY, "")
			constant.VERSION_KEY: codec.req.Version,
		}))
		if err := result.Error(); err != nil {
			if errRsp := sendErrorResp(header, []byte(err.Error())); errRsp != nil {
				log.Warn("Exporter: sendErrorResp(header:%#v, error:%v) = error:%s",
					header, err, errRsp)
				return jerrors.Trace(errRsp)
			}
		}
		if res := result.Result(); res != nil {
			rspStream, err := codec.Write("", res)
			if err != nil {
				return jerrors.Trace(err)
			}
			if errRsp := sendResp(header, rspStream); errRsp != nil {
				log.Warn("Exporter: sendResp(header:%#v, error:%v) = error:%s",
					header, err, errRsp)
				return jerrors.Trace(errRsp)
			}
		}
	}

	// get method
	svc := common.ServiceMap.GetService(JSONRPC, serviceName)
	if svc == nil {
		return jerrors.New("cannot find svc " + serviceName)
	}
	mtype := svc.Method()[methodName]
	if mtype == nil {
		return jerrors.New("cannot find method " + methodName + " of svc " + serviceName)
	}

	// get args
	var argv reflect.Value
	argIsValue := false
	if mtype.ArgType().Kind() == reflect.Ptr {
		argv = reflect.New(mtype.ArgType().Elem())
	} else {
		argv = reflect.New(mtype.ArgType())
		argIsValue = true
	}
	// argv guaranteed to be a pointer now.
	argvTmp := argv.Interface()
	argvTmp = args
	//if err = codec.ReadBody(argv.Interface()); err != nil {
	//	return jerrors.Trace(err)
	//}
	if argIsValue {
		argv = argv.Elem()
	}

	replyv := reflect.New(mtype.ReplyType().Elem())

	//  call service.method(args)
	var errMsg string
	returnValues := mtype.Method().Func.Call([]reflect.Value{
		svc.Rcvr(),
		mtype.SuiteContext(ctx),
		reflect.ValueOf(argvTmp),
		reflect.ValueOf(replyv.Interface()),
	})
	// The return value for the method is an error.
	if retErr := returnValues[0].Interface(); retErr != nil {
		errMsg = retErr.(error).Error()
	}

	// write response
	code := 200
	rspReply := replyv.Interface()
	if len(errMsg) != 0 {
		code = 500
		rspReply = invalidRequest
	}
	rspStream, err := codec.Write(errMsg, rspReply)
	if err != nil {
		return jerrors.Trace(err)
	}
	rsp := &http.Response{
		StatusCode:    code,
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        make(http.Header),
		ContentLength: int64(len(rspStream)),
		Body:          ioutil.NopCloser(bytes.NewReader(rspStream)),
	}
	delete(header, "Content-Type")
	delete(header, "Content-Length")
	delete(header, "Timeout")
	for k, v := range header {
		rsp.Header.Set(k, v)
	}

	rspBuf := bytes.NewBuffer(make([]byte, DefaultHTTPRspBufferSize))
	rspBuf.Reset()
	if err = rsp.Write(rspBuf); err != nil {
		log.Warn("rsp.Write(rsp:%#v) = error:%s", rsp, err)
		return nil
	}
	if _, err = rspBuf.WriteTo(conn); err != nil {
		log.Warn("rspBuf.WriteTo(conn:%#v) = error:%s", conn, err)
	}
	return nil
}
