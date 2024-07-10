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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"runtime"
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"google.golang.org/protobuf/types/known/anypb"
)

const (
	tripleUnaryHeaderCompression       = "Content-Encoding"
	tripleUnaryHeaderAcceptCompression = "Accept-Encoding"
	tripleUnaryTrailerPrefix           = "Trailer-"
	tripleHeaderTimeout                = "Triple-Timeout-Ms"
	tripleHeaderProtocolVersion        = "Triple-Protocol-Version"
	tripleProtocolVersion              = "0.1.0"

	tripleUnaryContentTypePrefix = "application/"
	tripleUnaryContentTypeJSON   = tripleUnaryContentTypePrefix + "json"
)

// defaultTripleUserAgent returns a User-Agent string similar to those used in gRPC.
//
//nolint:gochecknoglobals
var defaultTripleUserAgent = fmt.Sprintf("triple-go/%s (%s)", Version, runtime.Version())

type protocolTriple struct{}

// NewHandler implements protocol, so it must return an interface.
func (*protocolTriple) NewHandler(params *protocolHandlerParams) protocolHandler {
	if params.Spec.StreamType != StreamTypeUnary {
		panic("protocol triple does not support stream type rpc")
	}
	methods := make(map[string]struct{})
	// we decide not to support GET
	methods[http.MethodPost] = struct{}{}

	contentTypes := make(map[string]struct{})
	for _, name := range params.Codecs.Names() {
		contentTypes[canonicalizeContentType(tripleUnaryContentTypePrefix+name)] = struct{}{}
	}

	return &tripleHandler{
		protocolHandlerParams: *params,
		methods:               methods,
		accept:                contentTypes,
	}
}

// NewClient implements protocol, so it must return an interface.
func (*protocolTriple) NewClient(params *protocolClientParams) (protocolClient, error) {
	return &tripleClient{
		protocolClientParams: *params,
		peer:                 newPeerFromURL(params.URL, ProtocolTriple),
	}, nil
}

type tripleHandler struct {
	protocolHandlerParams

	methods map[string]struct{}
	accept  map[string]struct{}
}

func (h *tripleHandler) Methods() map[string]struct{} {
	return h.methods
}

func (h *tripleHandler) ContentTypes() map[string]struct{} {
	return h.accept
}

func (*tripleHandler) SetTimeout(request *http.Request) (context.Context, context.CancelFunc, error) {
	timeout := getHeaderCanonical(request.Header, tripleHeaderTimeout)
	if timeout == "" {
		return request.Context(), nil, nil
	}
	if len(timeout) > 10 {
		return nil, nil, errorf(CodeInvalidArgument, "parse timeout: %q has >10 digits", timeout)
	}
	millis, err := strconv.ParseInt(timeout, 10 /* base */, 64 /* bitsize */)
	if err != nil {
		return nil, nil, errorf(CodeInvalidArgument, "parse timeout: %w", err)
	}
	ctx, cancel := context.WithTimeout(
		request.Context(),
		time.Duration(millis)*time.Millisecond,
	)
	return ctx, cancel, nil
}

func (h *tripleHandler) CanHandlePayload(request *http.Request, contentType string) bool {
	_, ok := h.accept[contentType]
	return ok
}

func (h *tripleHandler) NewConn(
	responseWriter http.ResponseWriter,
	request *http.Request,
) (handlerConnCloser, bool) {
	if request.Method != http.MethodPost {
		return nil, false
	}
	// We need to parse metadata before entering the interceptor stack; we'll
	// send the error to the client later on.
	var contentEncoding, acceptEncoding string
	contentEncoding = getHeaderCanonical(request.Header, tripleUnaryHeaderCompression)
	acceptEncoding = getHeaderCanonical(request.Header, tripleUnaryHeaderAcceptCompression)
	requestCompression, responseCompression, failed := negotiateCompression(
		h.CompressionPools,
		contentEncoding,
		acceptEncoding,
	)
	if failed == nil {
		version := getHeaderCanonical(request.Header, tripleHeaderProtocolVersion)
		if version == "" && h.RequireTripleProtocolHeader {
			failed = errorf(CodeInvalidArgument, "missing required header: set %s to %q", tripleHeaderProtocolVersion, tripleProtocolVersion)
		} else if version != "" && version != tripleProtocolVersion {
			failed = errorf(CodeInvalidArgument, "%s must be %q: got %q", tripleHeaderProtocolVersion, tripleProtocolVersion, version)
		}
	}

	var requestBody io.ReadCloser
	var contentType, codecName string
	requestBody = request.Body
	//Prioritize codec specified by content-type
	contentType = getHeaderCanonical(request.Header, headerContentType)
	codecName = tripleCodecFromContentType(
		h.Spec.StreamType,
		contentType,
	)
	codec := h.Codecs.Get(codecName)
	backupCodec := h.Codecs.Get(h.ExpectedCodecName)
	// todo:// need to figure it out
	// The codec can be nil in the GET request case; that's okay: when failed
	// is non-nil, codec is never used.
	if failed == nil && codec == nil {
		failed = errorf(CodeInvalidArgument, "invalid message encoding: %q", codecName)
	}

	// Write any remaining headers here:
	// (1) any writes to the stream will implicitly send the headers, so we
	// should get all of gRPC's required response headers ready.
	// (2) interceptors should be able to see these headers.
	//
	// Since we know that these header keys are already in canonical form, we can
	// skip the normalization in Header.Set.
	header := responseWriter.Header()
	header[headerContentType] = []string{contentType}
	acceptCompressionHeader := tripleUnaryHeaderAcceptCompression
	header[acceptCompressionHeader] = []string{h.CompressionPools.CommaSeparatedNames()}

	var conn handlerConnCloser
	peer := Peer{
		Addr:     request.RemoteAddr,
		Protocol: ProtocolTriple,
	}
	conn = &tripleUnaryHandlerConn{
		spec:           h.Spec,
		peer:           peer,
		request:        request,
		responseWriter: responseWriter,
		marshaler: tripleUnaryMarshaler{
			writer:           responseWriter,
			codec:            codec,
			backupCodec:      backupCodec,
			compressMinBytes: h.CompressMinBytes,
			compressionName:  responseCompression,
			compressionPool:  h.CompressionPools.Get(responseCompression),
			bufferPool:       h.BufferPool,
			header:           responseWriter.Header(),
			sendMaxBytes:     h.SendMaxBytes,
		},
		unmarshaler: tripleUnaryUnmarshaler{
			reader:          requestBody,
			codec:           codec,
			backupCodec:     backupCodec,
			compressionPool: h.CompressionPools.Get(requestCompression),
			bufferPool:      h.BufferPool,
			readMaxBytes:    h.ReadMaxBytes,
		},
		responseTrailer: make(http.Header),
	}
	conn = wrapHandlerConnWithCodedErrors(conn)

	if failed != nil {
		// Negotiation failed, so we can't establish a stream.
		_ = conn.Close(failed)
		return nil, false
	}
	return conn, true
}

type tripleClient struct {
	protocolClientParams

	peer Peer
}

func (c *tripleClient) Peer() Peer {
	return c.peer
}

func (c *tripleClient) WriteRequestHeader(streamType StreamType, header http.Header) {
	// We know these header keys are in canonical form, so we can bypass all the
	// checks in Header.Set.
	if getHeaderCanonical(header, headerUserAgent) == "" {
		header[headerUserAgent] = []string{defaultTripleUserAgent}
	}
	header[tripleHeaderProtocolVersion] = []string{tripleProtocolVersion}
	header[headerContentType] = []string{
		tripleContentTypeFromCodecName(streamType, c.Codec.Name()),
	}
	if acceptCompression := c.CompressionPools.CommaSeparatedNames(); acceptCompression != "" {
		header[tripleUnaryHeaderAcceptCompression] = []string{acceptCompression}
	}
}

func (c *tripleClient) NewConn(
	ctx context.Context,
	spec Spec,
	header http.Header,
) StreamingClientConn {
	if deadline, ok := ctx.Deadline(); ok {
		millis := int64(time.Until(deadline) / time.Millisecond)
		if millis > 0 {
			encoded := strconv.FormatInt(millis, 10 /* base */)
			if len(encoded) <= 10 {
				header[tripleHeaderTimeout] = []string{encoded}
			} // else effectively unbounded
		}
	}
	duplexCall := newDuplexHTTPCall(ctx, c.HTTPClient, c.URL, spec, header)
	unaryConn := &tripleUnaryClientConn{
		spec:             spec,
		peer:             c.Peer(),
		duplexCall:       duplexCall,
		compressionPools: c.CompressionPools,
		bufferPool:       c.BufferPool,
		marshaler: tripleUnaryRequestMarshaler{
			tripleUnaryMarshaler: tripleUnaryMarshaler{
				writer:           duplexCall,
				codec:            c.Codec,
				compressMinBytes: c.CompressMinBytes,
				compressionName:  c.CompressionName,
				compressionPool:  c.CompressionPools.Get(c.CompressionName),
				bufferPool:       c.BufferPool,
				header:           duplexCall.Header(),
				sendMaxBytes:     c.SendMaxBytes,
			},
		},
		unmarshaler: tripleUnaryUnmarshaler{
			reader:       duplexCall,
			codec:        c.Codec,
			bufferPool:   c.BufferPool,
			readMaxBytes: c.ReadMaxBytes,
		},
		responseHeader:  make(http.Header),
		responseTrailer: make(http.Header),
	}
	duplexCall.SetValidateResponse(unaryConn.validateResponse)
	return wrapClientConnWithCodedErrors(unaryConn)
}

type tripleUnaryClientConn struct {
	spec             Spec
	peer             Peer
	duplexCall       *duplexHTTPCall
	compressionPools readOnlyCompressionPools
	bufferPool       *bufferPool
	marshaler        tripleUnaryRequestMarshaler
	unmarshaler      tripleUnaryUnmarshaler
	responseHeader   http.Header
	responseTrailer  http.Header
}

func (cc *tripleUnaryClientConn) Spec() Spec {
	return cc.spec
}

func (cc *tripleUnaryClientConn) Peer() Peer {
	return cc.peer
}

func (cc *tripleUnaryClientConn) Send(msg interface{}) error {
	if err := cc.marshaler.Marshal(msg); err != nil {
		return err
	}
	return nil // must be a literal nil: nil *Error is a non-nil error
}

func (cc *tripleUnaryClientConn) RequestHeader() http.Header {
	return cc.duplexCall.Header()
}

func (cc *tripleUnaryClientConn) CloseRequest() error {
	return cc.duplexCall.CloseWrite()
}

func (cc *tripleUnaryClientConn) Receive(msg interface{}) error {
	cc.duplexCall.BlockUntilResponseReady()
	if err := cc.unmarshaler.Unmarshal(msg); err != nil {
		return err
	}
	return nil // must be a literal nil: nil *Error is a non-nil error
}

func (cc *tripleUnaryClientConn) ResponseHeader() http.Header {
	cc.duplexCall.BlockUntilResponseReady()
	return cc.responseHeader
}

func (cc *tripleUnaryClientConn) ResponseTrailer() http.Header {
	cc.duplexCall.BlockUntilResponseReady()
	return cc.responseTrailer
}

func (cc *tripleUnaryClientConn) CloseResponse() error {
	return cc.duplexCall.CloseRead()
}

func (cc *tripleUnaryClientConn) validateResponse(response *http.Response) *Error {
	for k, v := range response.Header {
		if !strings.HasPrefix(k, tripleUnaryTrailerPrefix) {
			cc.responseHeader[k] = v
			continue
		}
		cc.responseTrailer[strings.TrimPrefix(k, tripleUnaryTrailerPrefix)] = v
	}
	compression := getHeaderCanonical(response.Header, tripleUnaryHeaderCompression)
	if compression != "" &&
		compression != compressionIdentity &&
		!cc.compressionPools.Contains(compression) {
		return errorf(
			CodeInternal,
			"unknown encoding %q: accepted encodings are %v",
			compression,
			cc.compressionPools.CommaSeparatedNames(),
		)
	}
	if response.StatusCode != http.StatusOK {
		unmarshaler := tripleUnaryUnmarshaler{
			reader:          response.Body,
			compressionPool: cc.compressionPools.Get(compression),
			bufferPool:      cc.bufferPool,
		}
		var wireErr tripleWireError
		if err := unmarshaler.UnmarshalFunc(&wireErr, json.Unmarshal); err != nil {
			return NewError(
				tripleHTTPToCode(response.StatusCode),
				errors.New(response.Status),
			)
		}
		serverErr := wireErr.asError()
		if serverErr == nil {
			return nil
		}
		serverErr.meta = cc.responseHeader.Clone()
		mergeHeaders(serverErr.meta, cc.responseTrailer)
		return serverErr
	}
	cc.unmarshaler.compressionPool = cc.compressionPools.Get(compression)
	return nil
}

type tripleUnaryHandlerConn struct {
	spec            Spec
	peer            Peer
	request         *http.Request
	responseWriter  http.ResponseWriter
	marshaler       tripleUnaryMarshaler
	unmarshaler     tripleUnaryUnmarshaler
	responseTrailer http.Header
	wroteBody       bool
}

func (hc *tripleUnaryHandlerConn) Spec() Spec {
	return hc.spec
}

func (hc *tripleUnaryHandlerConn) Peer() Peer {
	return hc.peer
}

func (hc *tripleUnaryHandlerConn) Receive(msg interface{}) error {
	if err := hc.unmarshaler.Unmarshal(msg); err != nil {
		return err
	}
	return nil // must be a literal nil: nil *Error is a non-nil error
}

func (hc *tripleUnaryHandlerConn) RequestHeader() http.Header {
	return hc.request.Header
}

func (hc *tripleUnaryHandlerConn) ExportableHeader() http.Header {
	// by now, there are no reserved headers
	return hc.request.Header
}

func (hc *tripleUnaryHandlerConn) Send(msg interface{}) error {
	hc.wroteBody = true
	hc.writeResponseHeader(nil /* error */)
	if err := hc.marshaler.Marshal(msg); err != nil {
		return err
	}
	return nil // must be a literal nil: nil *Error is a non-nil error
}

func (hc *tripleUnaryHandlerConn) ResponseHeader() http.Header {
	return hc.responseWriter.Header()
}

func (hc *tripleUnaryHandlerConn) ResponseTrailer() http.Header {
	return hc.responseTrailer
}

func (hc *tripleUnaryHandlerConn) Close(err error) error {
	if !hc.wroteBody {
		hc.writeResponseHeader(err)
	}
	if err == nil {
		return hc.request.Body.Close()
	}
	// In unary triple, errors always use application/json.
	setHeaderCanonical(hc.responseWriter.Header(), headerContentType, tripleUnaryContentTypeJSON)
	hc.responseWriter.WriteHeader(tripleCodeToHTTP(CodeOf(err)))
	data, marshalErr := json.Marshal(newTripleWireError(err))
	if marshalErr != nil {
		_ = hc.request.Body.Close()
		return errorf(CodeInternal, "marshal error: %w", err)
	}
	if _, writeErr := hc.responseWriter.Write(data); writeErr != nil {
		_ = hc.request.Body.Close()
		return writeErr
	}
	return hc.request.Body.Close()
}

func (hc *tripleUnaryHandlerConn) writeResponseHeader(err error) {
	header := hc.responseWriter.Header()
	if err != nil {
		if tripleErr, ok := asError(err); ok {
			mergeHeaders(header, tripleErr.meta)
		}
	}
	for k, v := range hc.responseTrailer {
		header[tripleUnaryTrailerPrefix+k] = v
	}
}

type tripleUnaryMarshaler struct {
	writer           io.Writer
	codec            Codec
	backupCodec      Codec
	compressMinBytes int
	compressionName  string
	compressionPool  *compressionPool
	bufferPool       *bufferPool
	header           http.Header
	sendMaxBytes     int
}

func (m *tripleUnaryMarshaler) Marshal(message interface{}) *Error {
	if message == nil {
		return m.write(nil)
	}
	data, err := m.codec.Marshal(message)
	if err != nil {
		if m.backupCodec != nil && m.codec.Name() != m.backupCodec.Name() {
			logger.Warnf("failed to marshal message with codec %s, trying alternative codec %s", m.codec.Name(), m.backupCodec.Name())
			data, err = m.backupCodec.Marshal(message)
		}
		if err != nil {
			return errorf(CodeInternal, "marshal message: %w", err)
		}
	}
	// Can't avoid allocating the slice, but we can reuse it.
	uncompressed := bytes.NewBuffer(data)
	defer m.bufferPool.Put(uncompressed)
	if len(data) < m.compressMinBytes || m.compressionPool == nil {
		if m.sendMaxBytes > 0 && len(data) > m.sendMaxBytes {
			return NewError(CodeResourceExhausted, fmt.Errorf("message size %d exceeds sendMaxBytes %d", len(data), m.sendMaxBytes))
		}
		return m.write(data)
	}
	compressed := m.bufferPool.Get()
	defer m.bufferPool.Put(compressed)
	if err := m.compressionPool.Compress(compressed, uncompressed); err != nil {
		return err
	}
	if m.sendMaxBytes > 0 && compressed.Len() > m.sendMaxBytes {
		return NewError(CodeResourceExhausted, fmt.Errorf("compressed message size %d exceeds sendMaxBytes %d", compressed.Len(), m.sendMaxBytes))
	}
	setHeaderCanonical(m.header, tripleUnaryHeaderCompression, m.compressionName)
	return m.write(compressed.Bytes())
}

func (m *tripleUnaryMarshaler) write(data []byte) *Error {
	if _, err := m.writer.Write(data); err != nil {
		if tripleErr, ok := asError(err); ok {
			return tripleErr
		}
		return errorf(CodeUnknown, "write message: %w", err)
	}
	return nil
}

type tripleUnaryRequestMarshaler struct {
	tripleUnaryMarshaler

	stableCodec stableCodec
	duplexCall  *duplexHTTPCall
}

func (m *tripleUnaryRequestMarshaler) Marshal(message interface{}) *Error {
	return m.tripleUnaryMarshaler.Marshal(message)
}

type tripleUnaryUnmarshaler struct {
	reader          io.Reader
	codec           Codec
	backupCodec     Codec //backupCodec is for the situation when content-type mismatches with the expected codec
	compressionPool *compressionPool
	bufferPool      *bufferPool
	alreadyRead     bool
	readMaxBytes    int
}

func (u *tripleUnaryUnmarshaler) Unmarshal(message interface{}) *Error {
	err := u.UnmarshalFunc(message, u.codec.Unmarshal)
	if err != nil {
		if u.backupCodec != nil && u.codec.Name() != u.backupCodec.Name() {
			logger.Warnf("failed to unmarshal message with codec %s, trying alternative codec %s", u.codec.Name(), u.backupCodec.Name())
			err = u.UnmarshalFunc(message, u.codec.Unmarshal)
		}
	}
	return err
}

func (u *tripleUnaryUnmarshaler) UnmarshalFunc(message interface{}, unmarshal func([]byte, interface{}) error) *Error {
	if u.alreadyRead {
		return NewError(CodeInternal, io.EOF)
	}
	u.alreadyRead = true
	data := u.bufferPool.Get()
	defer u.bufferPool.Put(data)
	reader := u.reader
	if u.readMaxBytes > 0 && int64(u.readMaxBytes) < math.MaxInt64 {
		reader = io.LimitReader(u.reader, int64(u.readMaxBytes)+1)
	}
	// ReadFrom ignores io.EOF, so any error here is real.
	// use io.LimitReader to prevent ReadFrom from panic
	bytesRead, err := data.ReadFrom(reader)
	if err != nil {
		if tripleErr, ok := asError(err); ok {
			return tripleErr
		}
		if readMaxBytesErr := asMaxBytesError(err, "read first %d bytes of message", bytesRead); readMaxBytesErr != nil {
			return readMaxBytesErr
		}
		return errorf(CodeUnknown, "read message: %w", err)
	}
	if u.readMaxBytes > 0 && bytesRead > int64(u.readMaxBytes) {
		// Attempt to read to end in order to allow connection re-use
		discardedBytes, err := io.Copy(io.Discard, u.reader)
		if err != nil {
			return errorf(CodeResourceExhausted, "message is larger than configured max %d - unable to determine message size: %w", u.readMaxBytes, err)
		}
		return errorf(CodeResourceExhausted, "message size %d is larger than configured max %d", bytesRead+discardedBytes, u.readMaxBytes)
	}
	if data.Len() > 0 && u.compressionPool != nil {
		decompressed := u.bufferPool.Get()
		defer u.bufferPool.Put(decompressed)
		if err := u.compressionPool.Decompress(decompressed, data, int64(u.readMaxBytes)); err != nil {
			return err
		}
		data = decompressed
	}
	if err := unmarshal(data.Bytes(), message); err != nil {
		return errorf(CodeInvalidArgument, "unmarshal into %T: %w", message, err)
	}
	return nil
}

type tripleWireDetail ErrorDetail

func (d *tripleWireDetail) MarshalJSON() ([]byte, error) {
	if d.wireJSON != "" {
		// If we unmarshaled this detail from JSON, return the original data. This
		// lets proxies w/o protobuf descriptors preserve human-readable details.
		return []byte(d.wireJSON), nil
	}
	wire := struct {
		Type  string          `json:"type"`
		Value string          `json:"value"`
		Debug json.RawMessage `json:"debug,omitempty"`
	}{
		Type:  strings.TrimPrefix(d.pb.TypeUrl, defaultAnyResolverPrefix),
		Value: base64.RawStdEncoding.EncodeToString(d.pb.Value),
	}
	// Try to produce debug info, but expect failure when we don't have
	// descriptors.
	var codec protoJSONCodec
	debug, err := codec.Marshal(d.pb)
	if err == nil && len(debug) > 2 { // don't bother sending `{}`
		wire.Debug = json.RawMessage(debug)
	}
	return json.Marshal(wire)
}

func (d *tripleWireDetail) UnmarshalJSON(data []byte) error {
	var wire struct {
		Type  string `json:"type"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}
	if !strings.Contains(wire.Type, "/") {
		wire.Type = defaultAnyResolverPrefix + wire.Type
	}
	decoded, err := DecodeBinaryHeader(wire.Value)
	if err != nil {
		return fmt.Errorf("decode base64: %w", err)
	}
	*d = tripleWireDetail{
		pb: &anypb.Any{
			TypeUrl: wire.Type,
			Value:   decoded,
		},
		wireJSON: string(data),
	}
	return nil
}

type tripleWireError struct {
	Code    Code                `json:"code"`
	Message string              `json:"message,omitempty"`
	Details []*tripleWireDetail `json:"details,omitempty"`
}

func newTripleWireError(err error) *tripleWireError {
	wire := &tripleWireError{
		Code:    CodeUnknown,
		Message: err.Error(),
	}
	if tripleErr, ok := asError(err); ok {
		wire.Code = tripleErr.Code()
		wire.Message = tripleErr.Message()
		if len(tripleErr.details) > 0 {
			wire.Details = make([]*tripleWireDetail, len(tripleErr.details))
			for i, detail := range tripleErr.details {
				wire.Details[i] = (*tripleWireDetail)(detail)
			}
		}
	}
	return wire
}

func (e *tripleWireError) asError() *Error {
	if e == nil {
		return nil
	}
	if e.Code < minCode || e.Code > maxCode {
		e.Code = CodeUnknown
	}
	err := NewWireError(e.Code, errors.New(e.Message))
	if len(e.Details) > 0 {
		err.details = make([]*ErrorDetail, len(e.Details))
		for i, detail := range e.Details {
			err.details[i] = (*ErrorDetail)(detail)
		}
	}
	return err
}

func tripleCodeToHTTP(code Code) int {
	// Return literals rather than named constants from the HTTP package to make
	// it easier to compare this function to the triple specification.
	switch code {
	case CodeCanceled:
		return 408
	case CodeUnknown:
		return 500
	case CodeInvalidArgument:
		return 400
	case CodeDeadlineExceeded:
		return 408
	case CodeNotFound:
		return 404
	case CodeAlreadyExists:
		return 409
	case CodePermissionDenied:
		return 403
	case CodeResourceExhausted:
		return 429
	case CodeFailedPrecondition:
		return 412
	case CodeAborted:
		return 409
	case CodeOutOfRange:
		return 400
	case CodeUnimplemented:
		return 404
	case CodeInternal:
		return 500
	case CodeUnavailable:
		return 503
	case CodeDataLoss:
		return 500
	case CodeUnauthenticated:
		return 401
	default:
		return 500 // same as CodeUnknown
	}
}

func tripleHTTPToCode(httpCode int) Code {
	// As above, literals are easier to compare to the specification (vs named
	// constants).
	switch httpCode {
	case 400:
		return CodeInvalidArgument
	case 401:
		return CodeUnauthenticated
	case 403:
		return CodePermissionDenied
	case 404:
		return CodeUnimplemented
	case 408:
		return CodeDeadlineExceeded
	case 412:
		return CodeFailedPrecondition
	case 413:
		return CodeResourceExhausted
	case 429:
		return CodeUnavailable
	case 431:
		return CodeResourceExhausted
	case 502, 503, 504:
		return CodeUnavailable
	default:
		return CodeUnknown
	}
}

func tripleCodecFromContentType(streamType StreamType, contentType string) string {
	if streamType == StreamTypeUnary {
		return strings.TrimPrefix(contentType, tripleUnaryContentTypePrefix)
	}
	return ""
}

func tripleContentTypeFromCodecName(streamType StreamType, name string) string {
	if streamType == StreamTypeUnary {
		return tripleUnaryContentTypePrefix + name
	}
	return ""
}
