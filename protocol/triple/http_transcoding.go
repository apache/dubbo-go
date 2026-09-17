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

package triple

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/grpc-ecosystem/grpc-gateway/v2/utilities"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/internal/httpbinding"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func (s *Server) httpTranscodingEnabled() bool {
	return s.cfg != nil && s.cfg.HTTPTranscoding != nil && s.cfg.HTTPTranscoding.Enabled
}

func (s *Server) buildHTTPTranscodingRoutes(interfaceName string, invoker base.Invoker, info *common.ServiceInfo) ([]tri.HTTPRoute, error) {
	if !s.httpTranscodingEnabled() || info == nil || invoker == nil {
		return nil, nil
	}

	// The service registration name may be a discovery/routing alias. Descriptor
	// lookup must always use the canonical protobuf service name carried by the
	// ServiceInfo, while the alias remains the canonical Triple route name.
	serviceName := strings.Trim(info.InterfaceName, "/")
	if serviceName == "" {
		serviceName = strings.Trim(interfaceName, "/")
	}
	if serviceName == "" {
		return nil, nil
	}

	url := invoker.GetURL()
	if url == nil {
		return nil, fmt.Errorf("cannot resolve HTTP transcoding routes for %q: invoker URL is nil", serviceName)
	}
	group, version := url.Group(), url.Version()
	maxBodyBytes := httpTranscodingMaxBodyBytes(s.cfg)
	routes := make([]tri.HTTPRoute, 0)
	for _, method := range info.Methods {
		rpcName := protoreflect.FullName(serviceName + "." + method.Name)
		bindings, err := httpbinding.Resolve(rpcName)
		if err != nil {
			if errors.Is(err, protoregistry.NotFound) {
				// Reflection/non-IDL service metadata can be used without a
				// registered protobuf descriptor. It keeps its existing Triple
				// route and simply contributes no REST route.
				continue
			}
			return nil, err
		}
		if method.Type != constant.CallUnary {
			// ResolveMethod validates annotated streaming methods and returns a
			// startup error. Unannotated streaming methods retain their normal
			// Triple registration and do not contribute an HTTP route.
			continue
		}
		for _, binding := range bindings {
			routes = append(routes, tri.HTTPRoute{
				Method:     binding.Method,
				Path:       binding.PathTemplate,
				RPC:        binding.RPC,
				Group:      group,
				Version:    version,
				PathFields: binding.PathFields,
				Handler:    newHTTPTranscodingHandler(binding, method, invoker, maxBodyBytes),
			})
		}
	}
	return routes, nil
}

func newHTTPTranscodingHandler(binding httpbinding.HTTPBinding, method common.MethodInfo, invoker base.Invoker, maxBodyBytes ...int64) runtime.HandlerFunc {
	var maxBody int64
	if len(maxBodyBytes) > 0 {
		maxBody = maxBodyBytes[0]
	}
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		request, err := newHTTPTranscodingRequest(method)
		if err != nil {
			writeHTTPTranscodingErrorResponse(w, err)
			return
		}
		bodyPath, err := canonicalHTTPTranscodingFieldPath(request.ProtoReflect().Descriptor(), binding.Body)
		if err != nil {
			writeHTTPTranscodingErrorResponse(w, tri.NewError(tri.CodeInternal, err))
			return
		}
		if err := decodeHTTPTranscodingBody(request, bodyPath, r, maxBody); err != nil {
			writeHTTPTranscodingErrorResponse(w, err)
			return
		}
		if err := populateHTTPTranscodingPath(request, binding.PathFields, pathParams); err != nil {
			writeHTTPTranscodingErrorResponse(w, tri.NewError(tri.CodeInvalidArgument, err))
			return
		}
		if err := populateHTTPTranscodingQuery(request, binding, r); err != nil {
			writeHTTPTranscodingErrorResponse(w, tri.NewError(tri.CodeInvalidArgument, err))
			return
		}

		ctx, outgoing := tri.NewHTTPTranscodingContext(r.Context(), r.Header)
		response, attachments, err := invokeUnaryRequest(ctx, method, invoker, request, r.Header)
		copyHTTPHeaders(w.Header(), outgoing.Header())
		copyHTTPHeaders(w.Header(), outgoing.Trailer())
		copyHTTPHeaders(w.Header(), tri.ExtractFromOutgoingContext(ctx))
		if err != nil {
			if response != nil {
				copyHTTPHeaders(w.Header(), response.Header())
				copyHTTPHeaders(w.Header(), response.Trailer())
			}
			copyHTTPAttachments(w.Header(), attachments)
			writeHTTPTranscodingErrorResponse(w, err)
			return
		}
		if response == nil {
			writeHTTPTranscodingErrorResponse(w, tri.NewError(tri.CodeInternal, fmt.Errorf("RPC %s returned a nil response", method.Name)))
			return
		}
		payload := responsePayload(response)
		if payload == nil {
			writeHTTPTranscodingErrorResponse(w, tri.NewError(tri.CodeInternal, fmt.Errorf("RPC %s returned a nil response payload", method.Name)))
			return
		}

		copyHTTPHeaders(w.Header(), response.Header())
		copyHTTPHeaders(w.Header(), response.Trailer())
		copyHTTPAttachments(w.Header(), attachments)
		body, err := marshalHTTPTranscodingResponse(payload, binding.ResponseBody)
		if err != nil {
			writeHTTPTranscodingErrorResponse(w, tri.NewError(tri.CodeInternal, err))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}
}

func newHTTPTranscodingRequest(method common.MethodInfo) (proto.Message, error) {
	if method.ReqInitFunc == nil {
		return nil, tri.NewError(tri.CodeInternal, fmt.Errorf("RPC %s has no request initializer", method.Name))
	}
	message, ok := method.ReqInitFunc().(proto.Message)
	if !ok || message == nil {
		return nil, tri.NewError(tri.CodeInternal, fmt.Errorf("RPC %s request is not a protobuf message", method.Name))
	}
	return message, nil
}

func decodeHTTPTranscodingBody(message proto.Message, body string, request *http.Request, maxBodyBytes int64) error {
	if body == "" || request.Body == nil {
		return nil
	}
	if contentType := strings.TrimSpace(request.Header.Get("Content-Type")); contentType != "" {
		mediaType, _, err := mime.ParseMediaType(contentType)
		if err != nil || (!strings.EqualFold(mediaType, "application/json") && !strings.EqualFold(mediaType, "application/json+protobuf")) {
			return &httpTranscodingHTTPError{
				status: http.StatusUnsupportedMediaType,
				err:    tri.NewError(tri.CodeInvalidArgument, fmt.Errorf("unsupported content type %q", contentType)),
			}
		}
	}

	raw, err := readHTTPTranscodingBody(request.Body, maxBodyBytes)
	if err != nil {
		return tri.NewError(tri.CodeInvalidArgument, fmt.Errorf("read request body: %w", err))
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil
	}
	if body != "*" {
		raw, err = wrapHTTPTranscodingBody(raw, body)
		if err != nil {
			return tri.NewError(tri.CodeInvalidArgument, err)
		}
	}
	var marshaler runtime.JSONPb
	if err := marshaler.Unmarshal(raw, message); err != nil {
		return tri.NewError(tri.CodeInvalidArgument, fmt.Errorf("decode request body: %w", err))
	}
	return nil
}

func readHTTPTranscodingBody(body io.Reader, maxBodyBytes int64) ([]byte, error) {
	if maxBodyBytes <= 0 {
		return io.ReadAll(body)
	}
	raw, err := io.ReadAll(io.LimitReader(body, maxBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(raw)) > maxBodyBytes {
		return nil, &httpTranscodingHTTPError{
			status: http.StatusRequestEntityTooLarge,
			err:    tri.NewError(tri.CodeResourceExhausted, fmt.Errorf("request body exceeds %d bytes", maxBodyBytes)),
		}
	}
	return raw, nil
}

func httpTranscodingMaxBodyBytes(cfg *global.TripleConfig) int64 {
	maxBodyBytes := int64(constant.DefaultMaxServerRecvMsgSize)
	if cfg == nil || cfg.MaxServerRecvMsgSize == "" {
		return maxBodyBytes
	}
	if parsed, err := humanize.ParseBytes(cfg.MaxServerRecvMsgSize); err == nil && parsed > 0 {
		maxBodyBytes = int64(parsed)
	}
	return maxBodyBytes
}

func wrapHTTPTranscodingBody(raw []byte, fieldPath string) ([]byte, error) {
	if !json.Valid(raw) {
		return nil, fmt.Errorf("request body is not valid JSON")
	}
	value := json.RawMessage(append([]byte(nil), raw...))
	parts := strings.Split(fieldPath, ".")
	for index := len(parts) - 1; index >= 0; index-- {
		if parts[index] == "" {
			return nil, fmt.Errorf("body field path %q contains an empty component", fieldPath)
		}
		wrapped, err := json.Marshal(map[string]json.RawMessage{parts[index]: value})
		if err != nil {
			return nil, fmt.Errorf("wrap request body for field %q: %w", fieldPath, err)
		}
		value = wrapped
	}
	return value, nil
}

func populateHTTPTranscodingPath(message proto.Message, fields []string, pathParams map[string]string) error {
	for _, field := range fields {
		value, ok := pathParams[field]
		if !ok {
			return fmt.Errorf("missing path parameter %q", field)
		}
		canonicalField, err := canonicalHTTPTranscodingFieldPath(message.ProtoReflect().Descriptor(), field)
		if err != nil {
			return err
		}
		if err := runtime.PopulateFieldFromPath(message, canonicalField, value); err != nil {
			return fmt.Errorf("populate path parameter %q: %w", field, err)
		}
	}
	return nil
}

func populateHTTPTranscodingQuery(message proto.Message, binding httpbinding.HTTPBinding, request *http.Request) error {
	// A wildcard body owns every request field. There are no remaining fields
	// that may be populated from the query string.
	if binding.Body == "*" {
		return nil
	}
	filterFields := make([][]string, 0, len(binding.PathFields)+1)
	for _, field := range binding.PathFields {
		canonicalField, err := canonicalHTTPTranscodingFieldPath(message.ProtoReflect().Descriptor(), field)
		if err != nil {
			return err
		}
		filterFields = append(filterFields, strings.Split(canonicalField, "."))
	}
	if binding.Body != "" {
		canonicalBody, err := canonicalHTTPTranscodingFieldPath(message.ProtoReflect().Descriptor(), binding.Body)
		if err != nil {
			return err
		}
		filterFields = append(filterFields, strings.Split(canonicalBody, "."))
	}
	if err := runtime.PopulateQueryParameters(message, request.URL.Query(), utilities.NewDoubleArray(filterFields)); err != nil {
		return fmt.Errorf("populate query parameters: %w", err)
	}
	return nil
}

func canonicalHTTPTranscodingFieldPath(message protoreflect.MessageDescriptor, path string) (string, error) {
	if path == "" || path == "*" {
		return path, nil
	}
	if message == nil {
		return "", fmt.Errorf("cannot resolve field path %q without a message descriptor", path)
	}
	parts := strings.Split(path, ".")
	canonical := make([]string, 0, len(parts))
	for index, part := range parts {
		if part == "" {
			return "", fmt.Errorf("field path %q contains an empty component", path)
		}
		field := message.Fields().ByName(protoreflect.Name(part))
		if field == nil {
			field = message.Fields().ByJSONName(part)
		}
		if field == nil {
			return "", fmt.Errorf("field %q not found in %q", part, message.FullName())
		}
		canonical = append(canonical, string(field.Name()))
		if index < len(parts)-1 {
			if field.IsList() || field.IsMap() || field.Message() == nil {
				return "", fmt.Errorf("field %q is not a singular message", field.FullName())
			}
			message = field.Message()
		}
	}
	return strings.Join(canonical, "."), nil
}

func invokeUnaryRequest(ctx context.Context, method common.MethodInfo, invoker base.Invoker, message any, header http.Header) (*tri.Response, map[string]any, error) {
	attachments := generateAttachments(header)
	ctx = context.WithValue(ctx, constant.AttachmentKey, attachments)
	invo := invocation.NewRPCInvocation(method.Name, extractUnaryInvocationArgs(message), attachments)
	res := invoker.Invoke(ctx, invo)
	if res == nil {
		return nil, nil, tri.NewError(tri.CodeInternal, fmt.Errorf("RPC %s returned a nil result", method.Name))
	}
	return wrapTripleResponse(res.Result()), res.Attachments(), res.Error()
}

func responsePayload(response *tri.Response) any {
	payload := response.Any()
	if values, ok := payload.([]any); ok {
		switch len(values) {
		case 0:
			return nil
		case 1:
			return values[0]
		}
	}
	return payload
}

func marshalHTTPTranscodingResponse(payload any, responseBody string) ([]byte, error) {
	var marshaler runtime.JSONPb
	if responseBody == "" || responseBody == "*" {
		return marshaler.Marshal(payload)
	}
	message, ok := payload.(proto.Message)
	if !ok || message == nil {
		return nil, fmt.Errorf("response_body %q requires a protobuf response", responseBody)
	}
	raw, err := marshaler.Marshal(message)
	if err != nil {
		return nil, err
	}
	return extractHTTPJSONField(raw, message.ProtoReflect().Descriptor(), responseBody)
}

func extractHTTPJSONField(raw []byte, descriptor protoreflect.MessageDescriptor, fieldPath string) ([]byte, error) {
	current := append([]byte(nil), raw...)
	parts := strings.Split(fieldPath, ".")
	for index, part := range parts {
		if part == "" {
			return nil, fmt.Errorf("response field path %q contains an empty component", fieldPath)
		}
		field := descriptor.Fields().ByName(protoreflect.Name(part))
		if field == nil {
			field = descriptor.Fields().ByJSONName(part)
		}
		if field == nil {
			return nil, fmt.Errorf("response field %q not found in %q", part, descriptor.FullName())
		}
		var object map[string]json.RawMessage
		if err := json.Unmarshal(current, &object); err != nil {
			return nil, fmt.Errorf("decode response field %q: %w", fieldPath, err)
		}
		value, ok := object[field.JSONName()]
		if !ok {
			value, ok = object[part]
		}
		if !ok {
			if index == len(parts)-1 {
				return defaultHTTPJSONField(field), nil
			}
			return []byte("null"), nil
		}
		current = append([]byte(nil), value...)
		if index < len(parts)-1 {
			if field.Message() == nil {
				return nil, fmt.Errorf("response field %q is not a message", field.FullName())
			}
			descriptor = field.Message()
		}
	}
	return current, nil
}

func defaultHTTPJSONField(field protoreflect.FieldDescriptor) []byte {
	if field.IsMap() {
		return []byte("{}")
	}
	if field.IsList() {
		return []byte("[]")
	}
	if field.Message() != nil {
		return []byte("null")
	}
	switch field.Kind() {
	case protoreflect.BoolKind:
		return []byte("false")
	case protoreflect.StringKind, protoreflect.BytesKind:
		return []byte(`""`)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind,
		protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return []byte(`"0"`)
	case protoreflect.EnumKind:
		if value := field.Enum().Values().ByNumber(0); value != nil {
			encoded, _ := json.Marshal(string(value.Name()))
			return encoded
		}
	}
	return []byte("0")
}

type httpTranscodingHTTPError struct {
	status int
	err    error
}

func (e *httpTranscodingHTTPError) Error() string {
	return e.err.Error()
}

func (e *httpTranscodingHTTPError) Unwrap() error {
	return e.err
}

func writeHTTPTranscodingErrorResponse(w http.ResponseWriter, err error) {
	if err == nil {
		err = tri.NewError(tri.CodeInternal, fmt.Errorf("HTTP transcoding failed"))
	}
	status := http.StatusInternalServerError
	if statusErr := new(httpTranscodingHTTPError); errors.As(err, &statusErr) {
		status = statusErr.status
		err = statusErr.err
	}

	code := tri.CodeOf(err)
	message := err.Error()
	var tripleErr *tri.Error
	details := []any{}
	if errors.As(err, &tripleErr) && tripleErr != nil {
		code = tripleErr.Code()
		message = tripleErr.Message()
		if message == "" {
			message = tripleErr.Code().String()
		}
		if status == http.StatusInternalServerError {
			status = httpStatusFromTripleCode(code)
		}
		copyHTTPHeaders(w.Header(), tripleErr.Meta())
		details = marshalHTTPTranscodingErrorDetails(tripleErr)
	} else if status == http.StatusInternalServerError {
		status = httpStatusFromTripleCode(code)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(httpTranscodingErrorBody{
		Code:    uint32(code),
		Message: message,
		Details: details,
	})
}

func marshalHTTPTranscodingErrorDetails(err *tri.Error) []any {
	if err == nil || len(err.Details()) == 0 {
		return []any{}
	}
	details := make([]any, 0, len(err.Details()))
	for _, detail := range err.Details() {
		if detail == nil {
			continue
		}
		value, valueErr := detail.Value()
		if valueErr != nil {
			details = append(details, map[string]any{"@type": "type.googleapis.com/" + detail.Type()})
			continue
		}
		var marshaler runtime.JSONPb
		raw, marshalErr := marshaler.Marshal(value)
		if marshalErr != nil {
			details = append(details, map[string]any{"@type": "type.googleapis.com/" + detail.Type()})
			continue
		}
		var object any
		if unmarshalErr := json.Unmarshal(raw, &object); unmarshalErr != nil {
			details = append(details, map[string]any{"@type": "type.googleapis.com/" + detail.Type()})
			continue
		}
		if fields, ok := object.(map[string]any); ok {
			fields["@type"] = "type.googleapis.com/" + detail.Type()
			details = append(details, fields)
			continue
		}
		details = append(details, map[string]any{
			"@type": "type.googleapis.com/" + detail.Type(),
			"value": object,
		})
	}
	return details
}

func httpStatusFromTripleCode(code tri.Code) int {
	switch code {
	case tri.CodeCanceled:
		return 499
	case tri.CodeInvalidArgument, tri.CodeFailedPrecondition, tri.CodeOutOfRange:
		return http.StatusBadRequest
	case tri.CodeDeadlineExceeded:
		return http.StatusGatewayTimeout
	case tri.CodeNotFound:
		return http.StatusNotFound
	case tri.CodeAlreadyExists, tri.CodeAborted:
		return http.StatusConflict
	case tri.CodePermissionDenied:
		return http.StatusForbidden
	case tri.CodeUnauthenticated:
		return http.StatusUnauthorized
	case tri.CodeResourceExhausted:
		return http.StatusTooManyRequests
	case tri.CodeUnimplemented:
		return http.StatusNotImplemented
	case tri.CodeUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

type httpTranscodingErrorBody struct {
	Code    uint32 `json:"code"`
	Message string `json:"message"`
	Details []any  `json:"details"`
}

func copyHTTPHeaders(dst, src http.Header) {
	for key, values := range src {
		for _, value := range values {
			dst.Add(key, value)
		}
	}
}

func copyHTTPAttachments(dst http.Header, attachments map[string]any) {
	for key, value := range attachments {
		switch value := value.(type) {
		case string:
			dst.Add(key, value)
		case []string:
			for _, item := range value {
				dst.Add(key, item)
			}
		}
	}
}
