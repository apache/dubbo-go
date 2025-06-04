// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triple_protocol

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

type TimeoutKey struct{}

// Client is a reusable, concurrency-safe client for a single procedure.
// Depending on the procedure's type, use the CallUnary, CallClientStream,
// CallServerStream, or CallBidiStream method.
//
// By default, clients use the gRPC protocol with the binary Protobuf Codec,
// ask for gzipped responses, and send uncompressed requests. To use the Triple,
// use the [WithTriple] options.
type Client struct {
	config         *clientConfig
	callUnary      func(context.Context, *Request, *Response) error
	protocolClient protocolClient
	err            error
}

// NewClient constructs a new Client.
func NewClient(httpClient HTTPClient, url string, options ...ClientOption) *Client {
	client := &Client{}
	config, err := newClientConfig(url, options)
	if err != nil {
		client.err = err
		return client
	}
	client.config = config
	protocolCli, protocolErr := client.config.Protocol.NewClient(
		&protocolClientParams{
			CompressionName: config.RequestCompressionName,
			CompressionPools: newReadOnlyCompressionPools(
				config.CompressionPools,
				config.CompressionNames,
			),
			Codec:            config.Codec,
			Protobuf:         config.protobuf(),
			CompressMinBytes: config.CompressMinBytes,
			HTTPClient:       httpClient,
			URL:              config.URL,
			BufferPool:       config.BufferPool,
			ReadMaxBytes:     config.ReadMaxBytes,
			SendMaxBytes:     config.SendMaxBytes,
			GetURLMaxBytes:   config.GetURLMaxBytes,
		},
	)
	if protocolErr != nil {
		client.err = protocolErr
		return client
	}
	client.protocolClient = protocolCli
	// Rather than applying unary interceptors along the hot path, we can do it
	// once at client creation.
	unarySpec := config.newSpec(StreamTypeUnary)
	unaryFunc := UnaryFunc(func(ctx context.Context, request AnyRequest, response AnyResponse) error {
		conn := client.protocolClient.NewConn(ctx, unarySpec, request.Header())
		// Send always returns an io.EOF unless the error is from the client-side.
		// We want the user to continue to call Receive in those cases to get the
		// full error from the server-side.
		logger.Errorf("here????????????????????")
		if err := conn.Send(request.Any()); err != nil && !errors.Is(err, io.EOF) {
			// for HTTP/1.1 case, CloseRequest must happen before CloseResponse
			// since HTTP/1.1 is of request-response type
			_ = conn.CloseRequest()
			_ = conn.CloseResponse()
			return err
		}
		if err := conn.CloseRequest(); err != nil {
			_ = conn.CloseResponse()
			return err
		}
		if err := receiveUnaryResponse(conn, response); err != nil {
			_ = conn.CloseResponse()
			return err
		}
		return conn.CloseResponse()
	})
	if interceptor := config.Interceptor; interceptor != nil {
		unaryFunc = interceptor.WrapUnary(unaryFunc)
	}
	client.callUnary = func(ctx context.Context, request *Request, response *Response) error {
		// To make the specification, peer, and RPC headers visible to the full
		// interceptor chain (as though they were supplied by the caller), we'll
		// add them here.
		request.spec = unarySpec
		request.peer = client.protocolClient.Peer()
		protocolCli.WriteRequestHeader(StreamTypeUnary, request.Header())
		if err := unaryFunc(ctx, request, response); err != nil {
			return err
		}

		return nil
	}
	return client
}

// CallUnary calls a request-response procedure.
func (c *Client) CallUnary(ctx context.Context, request *Request, response *Response) error {
	if c.err != nil {
		return c.err
	}
	ctx, flag, cancel := applyDefaultTimeout(ctx, c.config.Timeout)
	if flag {
		defer cancel()
	}
	mergeHeaders(request.Header(), ExtractFromOutgoingContext(ctx))
	applyGroupVersionHeaders(request.Header(), c.config)
	return c.callUnary(ctx, request, response)
}

// CallClientStream calls a client streaming procedure.
func (c *Client) CallClientStream(ctx context.Context) (*ClientStreamForClient, error) {
	if c.err != nil {
		return &ClientStreamForClient{err: c.err}, c.err
	}
	return &ClientStreamForClient{conn: c.newConn(ctx, StreamTypeClient)}, nil
}

// CallServerStream calls a server streaming procedure.
func (c *Client) CallServerStream(ctx context.Context, request *Request) (*ServerStreamForClient, error) {
	if c.err != nil {
		return nil, c.err
	}
	conn := c.newConn(ctx, StreamTypeServer)
	request.spec = conn.Spec()
	request.peer = conn.Peer()
	mergeHeaders(conn.RequestHeader(), request.header)
	// Send always returns an io.EOF unless the error is from the client-side.
	// We want the user to continue to call Receive in those cases to get the
	// full error from the server-side.
	if err := conn.Send(request.Msg); err != nil && !errors.Is(err, io.EOF) {
		_ = conn.CloseRequest()
		_ = conn.CloseResponse()
		return nil, err
	}
	if err := conn.CloseRequest(); err != nil {
		return nil, err
	}
	return &ServerStreamForClient{conn: conn}, nil
}

// CallBidiStream calls a bidirectional streaming procedure.
func (c *Client) CallBidiStream(ctx context.Context) (*BidiStreamForClient, error) {
	if c.err != nil {
		return &BidiStreamForClient{err: c.err}, c.err
	}
	return &BidiStreamForClient{conn: c.newConn(ctx, StreamTypeBidi)}, nil
}

func (c *Client) newConn(ctx context.Context, streamType StreamType) StreamingClientConn {
	newConn := func(ctx context.Context, spec Spec) StreamingClientConn {
		header := make(http.Header, 8) // arbitrary power of two, prevent immediate resizing
		mergeHeaders(header, ExtractFromOutgoingContext(ctx))
		applyGroupVersionHeaders(header, c.config)
		c.protocolClient.WriteRequestHeader(streamType, header)
		return c.protocolClient.NewConn(ctx, spec, header)
	}
	if interceptor := c.config.Interceptor; interceptor != nil {
		newConn = interceptor.WrapStreamingClient(newConn)
	}
	return newConn(ctx, c.config.newSpec(streamType))
}

type clientConfig struct {
	URL                    *url.URL
	Protocol               protocol
	Procedure              string
	CompressMinBytes       int
	Interceptor            Interceptor
	CompressionPools       map[string]*compressionPool
	CompressionNames       []string
	Codec                  Codec
	RequestCompressionName string
	BufferPool             *bufferPool
	ReadMaxBytes           int
	SendMaxBytes           int
	GetURLMaxBytes         int
	GetUseFallback         bool
	IdempotencyLevel       IdempotencyLevel
	Timeout                time.Duration
	Group                  string
	Version                string
}

func newClientConfig(rawURL string, options []ClientOption) (*clientConfig, *Error) {
	url, err := parseRequestURL(rawURL)
	if err != nil {
		return nil, err
	}
	protoPath := extractProtoPath(url.Path)
	config := clientConfig{
		URL: url,
		// use gRPC by default
		Protocol:         &protocolGRPC{},
		Procedure:        protoPath,
		CompressionPools: make(map[string]*compressionPool),
		BufferPool:       newBufferPool(),
	}
	// use proto binary by default
	withProtoBinaryCodec().applyToClient(&config)
	// use gzip by default
	withGzip().applyToClient(&config)
	for _, opt := range options {
		opt.applyToClient(&config)
	}
	if err := config.validate(); err != nil {
		return nil, err
	}
	return &config, nil
}

func (c *clientConfig) validate() *Error {
	if c.Codec == nil || c.Codec.Name() == "" {
		return errorf(CodeUnknown, "no codec configured")
	}
	if c.RequestCompressionName != "" && c.RequestCompressionName != compressionIdentity {
		if _, ok := c.CompressionPools[c.RequestCompressionName]; !ok {
			return errorf(CodeUnknown, "unknown compression %q", c.RequestCompressionName)
		}
	}
	return nil
}

func (c *clientConfig) protobuf() Codec {
	if c.Codec.Name() == codecNameProto {
		return c.Codec
	}
	return &protoBinaryCodec{}
}

func (c *clientConfig) newSpec(t StreamType) Spec {
	return Spec{
		StreamType:       t,
		Procedure:        c.Procedure,
		IsClient:         true,
		IdempotencyLevel: c.IdempotencyLevel,
	}
}

func parseRequestURL(rawURL string) (*url.URL, *Error) {
	url, err := url.ParseRequestURI(rawURL)
	if err == nil {
		return url, nil
	}
	if !strings.Contains(rawURL, "://") {
		// URL doesn't have a scheme, so the user is likely accustomed to
		// grpc-go's APIs.
		err = fmt.Errorf(
			"URL %q missing scheme: use http:// or https:// (unlike grpc-go)",
			rawURL,
		)
	}
	return nil, NewError(CodeUnavailable, err)
}

func applyDefaultTimeout(ctx context.Context, timeout time.Duration) (context.Context, bool, context.CancelFunc) {
	var cancel context.CancelFunc
	var applyFlag bool

	_, ok := ctx.Deadline()

	// Todo(finalt) Temporarily solve the problem that the timeout time is not valid
	if !ok {
		timeoutVal := ctx.Value(TimeoutKey{})
		if timeoutVal != nil {
			if s, exist := timeoutVal.(string); exist && s != "" {
				if newTimeout, err := time.ParseDuration(s); err == nil {
					ctx, cancel = context.WithDeadline(ctx, time.Now().Add(newTimeout))
					applyFlag = true
					return ctx, applyFlag, cancel
				}
			}
		}
	}

	if !ok && timeout != 0 {
		ctx, cancel = context.WithDeadline(ctx, time.Now().Add(timeout))
		applyFlag = true
	}
	return ctx, applyFlag, cancel
}

func applyGroupVersionHeaders(header http.Header, cfg *clientConfig) {
	if cfg.Group != "" {
		header.Set(tripleServiceGroup, cfg.Group)
	}
	if cfg.Version != "" {
		header.Set(tripleServiceVersion, cfg.Version)
	}
}
