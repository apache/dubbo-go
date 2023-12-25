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
	"fmt"
	"net/http"
)

const (
	defaultImplementationsSize = 4
)

// A Handler is the server-side implementation of a single RPC defined by a
// service schema.
//
// By default, Handlers support the Triple, gRPC, and gRPC-Web protocols with
// the binary Protobuf and JSON codecs. They support gzip compression using the
// standard library's [compress/gzip].
type Handler struct {
	spec Spec
	// key is group/version
	implementations  map[string]StreamingHandlerFunc
	protocolHandlers []protocolHandler
	allowMethod      string // Allow header
	acceptPost       string // Accept-Post header
}

// NewUnaryHandler constructs a [Handler] for a request-response procedure.
func NewUnaryHandler(
	procedure string,
	reqInitFunc func() interface{},
	unary func(context.Context, *Request) (*Response, error),
	options ...HandlerOption,
) *Handler {
	config := newHandlerConfig(procedure, options)
	implementation := generateUnaryHandlerFunc(procedure, reqInitFunc, unary, config.Interceptor)
	protocolHandlers := config.newProtocolHandlers(StreamTypeUnary)

	hdl := &Handler{
		spec:             config.newSpec(StreamTypeUnary),
		implementations:  make(map[string]StreamingHandlerFunc, defaultImplementationsSize),
		protocolHandlers: protocolHandlers,
		allowMethod:      sortedAllowMethodValue(protocolHandlers),
		acceptPost:       sortedAcceptPostValue(protocolHandlers),
	}
	hdl.processImplementation(getIdentifier(config.Group, config.Version), implementation)
	return hdl
}

func generateUnaryHandlerFunc(
	procedure string,
	reqInitFunc func() interface{},
	unary func(context.Context, *Request) (*Response, error),
	interceptor Interceptor,
) StreamingHandlerFunc {
	// Wrap the strongly-typed implementation so we can apply interceptors.
	untyped := UnaryHandlerFunc(func(ctx context.Context, request AnyRequest) (AnyResponse, error) {
		// verify err
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		typed, ok := request.(*Request)
		if !ok {
			return nil, errorf(CodeInternal, "unexpected handler request type %T", request)
		}
		res, err := unary(ctx, typed)
		if res == nil && err == nil {
			// This is going to panic during serialization. Debugging is much easier
			// if we panic here instead, so we can include the procedure name.
			panic(fmt.Sprintf("%s returned nil *triple.Response and nil error", procedure)) //nolint: forbidigo
		}
		return res, err
	})
	// todo: modify server func
	if interceptor != nil {
		untyped = interceptor.WrapUnaryHandler(untyped)
	}
	// receive and send
	// conn should be responsible for marshal and unmarshal
	// Given a stream, how should we call the unary function?
	implementation := func(ctx context.Context, conn StreamingHandlerConn) error {
		req := reqInitFunc()
		if err := conn.Receive(req); err != nil {
			return err
		}
		// wrap the specific msg
		request := NewRequest(req)
		request.spec = conn.Spec()
		request.peer = conn.Peer()
		request.header = conn.RequestHeader()
		// embed header in context so that user logic could process them via FromIncomingContext
		ctx = newIncomingContext(ctx, conn.RequestHeader())

		response, err := untyped(ctx, request)
		if err != nil {
			return err
		}
		// merge headers
		mergeHeaders(conn.ResponseHeader(), response.Header())
		mergeHeaders(conn.ResponseTrailer(), response.Trailer())
		return conn.Send(response.Any())
	}

	return implementation
}

// NewClientStreamHandler constructs a [Handler] for a client streaming procedure.
func NewClientStreamHandler(
	procedure string,
	streamFunc func(context.Context, *ClientStream) (*Response, error),
	options ...HandlerOption,
) *Handler {
	config := newHandlerConfig(procedure, options)
	implementation := generateClientStreamHandlerFunc(procedure, streamFunc, config.Interceptor)
	protocolHandlers := config.newProtocolHandlers(StreamTypeClient)

	hdl := &Handler{
		spec:             config.newSpec(StreamTypeClient),
		implementations:  make(map[string]StreamingHandlerFunc, defaultImplementationsSize),
		protocolHandlers: protocolHandlers,
		allowMethod:      sortedAllowMethodValue(protocolHandlers),
		acceptPost:       sortedAcceptPostValue(protocolHandlers),
	}
	hdl.processImplementation(getIdentifier(config.Group, config.Version), implementation)

	return hdl
}

func generateClientStreamHandlerFunc(
	procedure string,
	streamFunc func(context.Context, *ClientStream) (*Response, error),
	interceptor Interceptor,
) StreamingHandlerFunc {
	implementation := func(ctx context.Context, conn StreamingHandlerConn) error {
		stream := &ClientStream{conn: conn}
		// embed header in context so that user logic could process them via FromIncomingContext
		ctx = newIncomingContext(ctx, conn.RequestHeader())
		res, err := streamFunc(ctx, stream)
		if err != nil {
			return err
		}
		if res == nil {
			// This is going to panic during serialization. Debugging is much easier
			// if we panic here instead, so we can include the procedure name.
			panic(fmt.Sprintf("%s returned nil *triple.Response and nil error", procedure)) //nolint: forbidigo
		}
		mergeHeaders(conn.ResponseHeader(), res.header)
		mergeHeaders(conn.ResponseTrailer(), res.trailer)
		return conn.Send(res.Msg)
	}
	if interceptor != nil {
		implementation = interceptor.WrapStreamingHandler(implementation)
	}

	return implementation
}

// NewServerStreamHandler constructs a [Handler] for a server streaming procedure.
func NewServerStreamHandler(
	procedure string,
	reqInitFunc func() interface{},
	streamFunc func(context.Context, *Request, *ServerStream) error,
	options ...HandlerOption,
) *Handler {
	config := newHandlerConfig(procedure, options)
	implementation := generateServerStreamHandlerFunc(procedure, reqInitFunc, streamFunc, config.Interceptor)
	protocolHandlers := config.newProtocolHandlers(StreamTypeServer)

	hdl := &Handler{
		spec:             config.newSpec(StreamTypeServer),
		implementations:  make(map[string]StreamingHandlerFunc, defaultImplementationsSize),
		protocolHandlers: protocolHandlers,
		allowMethod:      sortedAllowMethodValue(protocolHandlers),
		acceptPost:       sortedAcceptPostValue(protocolHandlers),
	}
	hdl.processImplementation(getIdentifier(config.Group, config.Version), implementation)

	return hdl
}

func generateServerStreamHandlerFunc(
	procedure string,
	reqInitFunc func() interface{},
	streamFunc func(context.Context, *Request, *ServerStream) error,
	interceptor Interceptor,
) StreamingHandlerFunc {
	implementation := func(ctx context.Context, conn StreamingHandlerConn) error {
		req := reqInitFunc()
		if err := conn.Receive(req); err != nil {
			return err
		}
		// embed header in context so that user logic could process them via FromIncomingContext
		ctx = newIncomingContext(ctx, conn.RequestHeader())
		return streamFunc(
			ctx,
			&Request{
				Msg:    req,
				spec:   conn.Spec(),
				peer:   conn.Peer(),
				header: conn.RequestHeader(),
			},
			&ServerStream{conn: conn},
		)
	}
	if interceptor != nil {
		implementation = interceptor.WrapStreamingHandler(implementation)
	}

	return implementation
}

// NewBidiStreamHandler constructs a [Handler] for a bidirectional streaming procedure.
func NewBidiStreamHandler(
	procedure string,
	streamFunc func(context.Context, *BidiStream) error,
	options ...HandlerOption,
) *Handler {
	config := newHandlerConfig(procedure, options)
	implementation := generateBidiStreamHandlerFunc(procedure, streamFunc, config.Interceptor)
	protocolHandlers := config.newProtocolHandlers(StreamTypeBidi)

	hdl := &Handler{
		spec:             config.newSpec(StreamTypeBidi),
		implementations:  make(map[string]StreamingHandlerFunc, defaultImplementationsSize),
		protocolHandlers: protocolHandlers,
		allowMethod:      sortedAllowMethodValue(protocolHandlers),
		acceptPost:       sortedAcceptPostValue(protocolHandlers),
	}
	hdl.processImplementation(getIdentifier(config.Group, config.Version), implementation)

	return hdl
}

func generateBidiStreamHandlerFunc(
	procedure string,
	streamFunc func(context.Context, *BidiStream) error,
	interceptor Interceptor,
) StreamingHandlerFunc {
	implementation := func(ctx context.Context, conn StreamingHandlerConn) error {
		// embed header in context so that user logic could process them via FromIncomingContext
		ctx = newIncomingContext(ctx, conn.RequestHeader())
		return streamFunc(
			ctx,
			&BidiStream{conn: conn},
		)
	}
	if interceptor != nil {
		implementation = interceptor.WrapStreamingHandler(implementation)
	}

	return implementation
}

func (h *Handler) processImplementation(identifier string, implementation StreamingHandlerFunc) {
	h.implementations[identifier] = implementation
}

// ServeHTTP implements [http.Handler].
func (h *Handler) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	// We don't need to defer functions  to close the request body or read to
	// EOF: the stream we construct later on already does that, and we only
	// return early when dealing with misbehaving clients. In those cases, it's
	// okay if we can't re-use the connection.
	isBidi := (h.spec.StreamType & StreamTypeBidi) == StreamTypeBidi
	if isBidi && request.ProtoMajor < 2 {
		// Clients coded to expect full-duplex connections may hang if they've
		// mistakenly negotiated HTTP/1.1. To unblock them, we must close the
		// underlying TCP connection.
		responseWriter.Header().Set("Connection", "close")
		responseWriter.WriteHeader(http.StatusHTTPVersionNotSupported)
		return
	}

	// inspect headers
	var protocolHandlers []protocolHandler
	for _, handler := range h.protocolHandlers {
		if _, ok := handler.Methods()[request.Method]; ok {
			protocolHandlers = append(protocolHandlers, handler)
		}
	}

	if len(protocolHandlers) == 0 {
		responseWriter.Header().Set("Allow", h.allowMethod)
		responseWriter.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	contentType := canonicalizeContentType(getHeaderCanonical(request.Header, headerContentType))

	// inspect contentType
	// Find our implementation of the RPC protocol in use.
	var protocolHdl protocolHandler
	for _, handler := range protocolHandlers {
		if handler.CanHandlePayload(request, contentType) {
			protocolHdl = handler
			break
		}
	}
	if protocolHdl == nil {
		responseWriter.Header().Set("Accept-Post", h.acceptPost)
		responseWriter.WriteHeader(http.StatusUnsupportedMediaType)
		return
	}

	// Establish a stream and serve the RPC.
	setHeaderCanonical(request.Header, headerContentType, contentType)
	// process context
	ctx, cancel, timeoutErr := protocolHdl.SetTimeout(request) //nolint: contextcheck
	if timeoutErr != nil {
		ctx = request.Context()
	}
	if cancel != nil {
		defer cancel()
	}
	// create stream
	connCloser, ok := protocolHdl.NewConn(
		responseWriter,
		request.WithContext(ctx),
	)
	if !ok {
		// Failed to create stream, usually because client used an unknown
		// compression algorithm. Nothing further to do.
		return
	}
	if timeoutErr != nil {
		_ = connCloser.Close(timeoutErr)
		return
	}

	// invoke implementation
	svcGroup := request.Header.Get(tripleServiceGroup)
	svcVersion := request.Header.Get(tripleServiceVersion)
	// todo(DMwangnima): inspect ok
	implementation := h.implementations[getIdentifier(svcGroup, svcVersion)]
	_ = connCloser.Close(implementation(ctx, connCloser))
}

type handlerConfig struct {
	CompressionPools            map[string]*compressionPool
	CompressionNames            []string
	Codecs                      map[string]Codec
	CompressMinBytes            int
	Interceptor                 Interceptor
	Procedure                   string
	HandleGRPC                  bool
	RequireTripleProtocolHeader bool
	IdempotencyLevel            IdempotencyLevel
	BufferPool                  *bufferPool
	ReadMaxBytes                int
	SendMaxBytes                int
	Group                       string
	Version                     string
}

func newHandlerConfig(procedure string, options []HandlerOption) *handlerConfig {
	protoPath := extractProtoPath(procedure)
	config := handlerConfig{
		Procedure:        protoPath,
		CompressionPools: make(map[string]*compressionPool),
		Codecs:           make(map[string]Codec),
		HandleGRPC:       true,
		BufferPool:       newBufferPool(),
	}
	withProtoBinaryCodec().applyToHandler(&config)
	withProtoJSONCodecs().applyToHandler(&config)
	withHessian2Codec().applyToHandler(&config)
	withGzip().applyToHandler(&config)
	for _, opt := range options {
		opt.applyToHandler(&config)
	}
	return &config
}

func (c *handlerConfig) newSpec(streamType StreamType) Spec {
	return Spec{
		Procedure:        c.Procedure,
		StreamType:       streamType,
		IdempotencyLevel: c.IdempotencyLevel,
	}
}

func (c *handlerConfig) newProtocolHandlers(streamType StreamType) []protocolHandler {
	// initialize protocol
	var protocols []protocol
	if streamType == StreamTypeUnary {
		protocols = append(protocols, &protocolTriple{})
	}
	if c.HandleGRPC {
		protocols = append(protocols, &protocolGRPC{})
	}
	// protocol -> protocolHandler
	handlers := make([]protocolHandler, 0, len(protocols))
	// initialize codec and compressor
	codecs := newReadOnlyCodecs(c.Codecs)
	compressors := newReadOnlyCompressionPools(
		c.CompressionPools,
		c.CompressionNames,
	)
	for _, protocol := range protocols {
		handlers = append(handlers, protocol.NewHandler(&protocolHandlerParams{
			Spec:             c.newSpec(streamType),
			Codecs:           codecs,
			CompressionPools: compressors,
			// config content
			CompressMinBytes:            c.CompressMinBytes,
			BufferPool:                  c.BufferPool,
			ReadMaxBytes:                c.ReadMaxBytes,
			SendMaxBytes:                c.SendMaxBytes,
			RequireTripleProtocolHeader: c.RequireTripleProtocolHeader,
			IdempotencyLevel:            c.IdempotencyLevel,
		}))
	}
	return handlers
}

func getIdentifier(group, version string) string {
	return group + "/" + version
}

func newStreamHandler(
	procedure string,
	streamType StreamType,
	implementation StreamingHandlerFunc,
	options ...HandlerOption,
) *Handler {
	config := newHandlerConfig(procedure, options)
	if ic := config.Interceptor; ic != nil {
		implementation = ic.WrapStreamingHandler(implementation)
	}
	protocolHandlers := config.newProtocolHandlers(streamType)

	hdl := &Handler{
		spec:             config.newSpec(streamType),
		implementations:  make(map[string]StreamingHandlerFunc, defaultImplementationsSize),
		protocolHandlers: protocolHandlers,
		allowMethod:      sortedAllowMethodValue(protocolHandlers),
		acceptPost:       sortedAcceptPostValue(protocolHandlers),
	}
	hdl.processImplementation(getIdentifier(config.Group, config.Version), implementation)

	return hdl
}
