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

package triple_protocol

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"

	perrors "github.com/pkg/errors"

	msgpack "github.com/ugorji/go/codec"

	"google.golang.org/protobuf/encoding/protojson"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/runtime/protoiface"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/interoperability"
)

const (
	codecNameProto           = "proto"
	codecNameJSON            = "json"
	codecNameHessian2        = "hessian2"
	codecNameMsgPack         = "msgpack"
	codecNameJSONCharsetUTF8 = codecNameJSON + "; charset=utf-8"
)

// Codec marshals structs (typically generated from a schema) to and from bytes.
type Codec interface {
	// Name returns the name of the Codec.
	//
	// This may be used as part of the Content-Type within HTTP. For example,
	// with gRPC this is the content subtype, so "application/grpc+proto" will
	// map to the Codec with name "proto".
	//
	// Names must not be empty.
	Name() string
	// Marshal marshals the given message.
	//
	// Marshal may expect a specific type of message, and will error if this type
	// is not given.
	Marshal(any) ([]byte, error)
	// Unmarshal unmarshals the given message.
	//
	// Unmarshal may expect a specific type of message, and will error if this
	// type is not given.
	Unmarshal([]byte, any) error
}

// stableCodec is an extension to Codec for serializing with stable output.
type stableCodec interface {
	Codec

	// MarshalStable marshals the given message with stable field ordering.
	//
	// MarshalStable should return the same output for a given input. Although
	// it is not guaranteed to be canonicalized, the marshaling routine for
	// MarshalStable will opt for the most normalized output available for a
	// given serialization.
	//
	// For practical reasons, it is possible for MarshalStable to return two
	// different results for two inputs considered to be "equal" in their own
	// domain, and it may change in the future with codec updates, but for
	// any given concrete value and any given version, it should return the
	// same output.
	MarshalStable(any) ([]byte, error)

	// IsBinary returns true if the marshaled data is binary for this codec.
	//
	// If this function returns false, the data returned from Marshal and
	// MarshalStable are considered valid text and may be used in contexts
	// where text is expected.
	IsBinary() bool
}

// protoBinaryCodec handles standard protobuf binary serialization for IDL
// calls. Non-IDL (Java Dubbo Triple generic call) wrapper handling on the
// server side is handled by tripleServerCodecSession, which delegates to
// this codec for the IDL path.
type protoBinaryCodec struct{}

var _ Codec = (*protoBinaryCodec)(nil)

func (c *protoBinaryCodec) Name() string { return codecNameProto }

func (c *protoBinaryCodec) Marshal(message any) ([]byte, error) {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return nil, errNotProto(message)
	}
	return proto.Marshal(protoMessage)
}

func (c *protoBinaryCodec) Unmarshal(data []byte, message any) error {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return errNotProto(message)
	}
	return proto.Unmarshal(data, protoMessage)
}

func (c *protoBinaryCodec) MarshalStable(message any) ([]byte, error) {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return nil, errNotProto(message)
	}
	// protobuf does not offer a canonical output today, so this format is not
	// guaranteed to match deterministic output from other protobuf libraries.
	// In addition, unknown fields may cause inconsistent output for otherwise
	// equal messages.
	// https://github.com/golang/protobuf/issues/1121
	options := proto.MarshalOptions{Deterministic: true}
	return options.Marshal(protoMessage)
}

func (c *protoBinaryCodec) IsBinary() bool {
	return true
}

type protoJSONCodec struct {
	name string
}

var _ Codec = (*protoJSONCodec)(nil)

func (c *protoJSONCodec) Name() string { return c.name }

func (c *protoJSONCodec) Marshal(message any) ([]byte, error) {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return nil, errNotProto(message)
	}
	var options = protojson.MarshalOptions{
		UseProtoNames: true,
	}
	return options.Marshal(protoMessage)
}

func (c *protoJSONCodec) Unmarshal(binary []byte, message any) error {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return errNotProto(message)
	}
	if len(binary) == 0 {
		return errors.New("zero-length payload is not a valid JSON object")
	}
	var options protojson.UnmarshalOptions
	return options.Unmarshal(binary, protoMessage)
}

func (c *protoJSONCodec) MarshalStable(message any) ([]byte, error) {
	// protojson does not offer a "deterministic" field ordering, but fields
	// are still ordered consistently by their index. However, protojson can
	// output inconsistent whitespace for some reason, therefore it is
	// suggested to use a formatter to ensure consistent formatting.
	// https://github.com/golang/protobuf/issues/1373
	messageJSON, err := c.Marshal(message)
	if err != nil {
		return nil, err
	}
	compactedJSON := bytes.NewBuffer(messageJSON[:0])
	if err = json.Compact(compactedJSON, messageJSON); err != nil {
		return nil, err
	}
	return compactedJSON.Bytes(), nil
}

func (c *protoJSONCodec) IsBinary() bool {
	return false
}

// WrapperCodec is an interface for codecs that use a protobuf wrapper format
// (TripleRequestWrapper/TripleResponseWrapper) on the wire. This is required for
// interoperability with Java Dubbo Triple protocol in non-IDL mode.
//
// Codecs implementing this interface:
// - Use protobuf as the wire format (Content-Type: application/proto)
// - Wrap data in TripleRequestWrapper (for requests) or TripleResponseWrapper (for responses)
// - Use an inner codec (e.g., hessian2) for the actual data serialization
type WrapperCodec interface {
	Codec
	// WireCodecName returns "proto" because the wire format is protobuf.
	WireCodecName() string
}

// getWireCodecName returns the codec name to use for Content-Type on the wire.
// If the codec implements WrapperCodec, its WireCodecName() is used.
// Otherwise, the codec's Name() is used.
func getWireCodecName(codec Codec) string {
	if wrapper, ok := codec.(WrapperCodec); ok {
		return wrapper.WireCodecName()
	}
	return codec.Name()
}

// protoWrapperCodec wraps an inner codec (e.g., hessian2) in protobuf wrapper format.
// This is used for interoperability with Java Dubbo Triple protocol in non-IDL mode.
//
// Wire format:
//   - Requests use TripleRequestWrapper (multiple args, argTypes)
//   - Responses use TripleResponseWrapper (single data field)
//
// The Content-Type is "application/proto" because the outer format is protobuf.
// The inner serialization type (e.g., "hessian2") is stored in the wrapper's serializeType field.
type protoWrapperCodec struct {
	innerCodec Codec
}

var _ WrapperCodec = (*protoWrapperCodec)(nil)

// Name returns the inner codec name (e.g., "hessian2") for codec registration and lookup.
func (c *protoWrapperCodec) Name() string {
	return c.innerCodec.Name()
}

// WireCodecName returns "proto" because the wire format is protobuf.
// This ensures the correct Content-Type (application/proto) is used.
func (c *protoWrapperCodec) WireCodecName() string {
	return codecNameProto
}

// Marshal wraps the message in TripleRequestWrapper format for requests.
func (c *protoWrapperCodec) Marshal(message any) ([]byte, error) {
	reqs, ok := message.([]any)
	if !ok {
		reqs = []any{message}
	}

	reqsLen := len(reqs)
	reqsBytes := make([][]byte, reqsLen)
	reqsTypes := make([]string, reqsLen)
	for i, req := range reqs {
		reqBytes, err := c.innerCodec.Marshal(req)
		if err != nil {
			return nil, err
		}
		reqsBytes[i] = reqBytes
		reqsTypes[i] = getArgType(req)
	}

	wrapperReq := &interoperability.TripleRequestWrapper{
		SerializeType: c.innerCodec.Name(),
		Args:          reqsBytes,
		ArgTypes:      reqsTypes,
	}

	return proto.Marshal(wrapperReq)
}

// Unmarshal handles both TripleResponseWrapper (for responses) and TripleRequestWrapper (for requests).
// It determines the format by checking if message is a slice (request) or not (response).
func (c *protoWrapperCodec) Unmarshal(binary []byte, message any) error {
	// Check if message is a slice - if so, it's a request with multiple args
	if params, isSlice := message.([]any); isSlice {
		// Request format: TripleRequestWrapper with multiple args
		var wrapperReq interoperability.TripleRequestWrapper
		if err := proto.Unmarshal(binary, &wrapperReq); err != nil {
			return err
		}
		if len(wrapperReq.Args) != len(params) {
			return fmt.Errorf("wrapper codec: expected %d params, got %d args", len(params), len(wrapperReq.Args))
		}

		inner, err := resolveInnerCodec(wrapperReq.SerializeType)
		if err != nil {
			return fmt.Errorf("wrapper codec: %w", err)
		}
		for i, arg := range wrapperReq.Args {
			if err := inner.Unmarshal(arg, params[i]); err != nil {
				return err
			}
		}
		return nil
	}

	// Response format: TripleResponseWrapper with single data field
	var wrapperResp interoperability.TripleResponseWrapper
	if err := proto.Unmarshal(binary, &wrapperResp); err == nil {
		inner, err := resolveInnerCodec(wrapperResp.SerializeType)
		if err != nil {
			return fmt.Errorf("wrapper codec: %w", err)
		}
		// Non-empty Data: decode the single return value.
		if len(wrapperResp.Data) > 0 {
			return inner.Unmarshal(wrapperResp.Data, message)
		}
		// Empty Data with a validated SerializeType is a null/void response.
		return nil
	}

	// Fallback: try as single-arg request (not a response wrapper)
	var wrapperReq interoperability.TripleRequestWrapper
	if err := proto.Unmarshal(binary, &wrapperReq); err != nil {
		return fmt.Errorf("wrapper codec: failed to unmarshal as request or response wrapper")
	}
	if len(wrapperReq.Args) != 1 {
		return fmt.Errorf("wrapper codec: expected 1 arg for single param, got %d", len(wrapperReq.Args))
	}
	inner, err := resolveInnerCodec(wrapperReq.SerializeType)
	if err != nil {
		return fmt.Errorf("wrapper codec: %w", err)
	}
	return inner.Unmarshal(wrapperReq.Args[0], message)
}

func newProtoWrapperCodec(innerCodec Codec) *protoWrapperCodec {
	return &protoWrapperCodec{innerCodec: innerCodec}
}

// todo(DMwangnima): add unit tests
type hessian2Codec struct{}

func (h *hessian2Codec) Name() string {
	return codecNameHessian2
}

func (c *hessian2Codec) Marshal(message any) ([]byte, error) {
	encoder := hessian.NewEncoder()
	if err := encoder.Encode(message); err != nil {
		return nil, err
	}

	return encoder.Buffer(), nil
}

func (c *hessian2Codec) Unmarshal(binary []byte, message any) error {
	decoder := hessian.NewDecoder(binary)
	val, err := decoder.Decode()
	if err != nil {
		return err
	}
	return reflectResponse(val, message)
}

type msgpackCodec struct{}

func (c *msgpackCodec) Name() string {
	return codecNameMsgPack
}

func (c *msgpackCodec) Marshal(message any) ([]byte, error) {
	var out []byte
	encoder := msgpack.NewEncoderBytes(&out, new(msgpack.MsgpackHandle))
	return out, encoder.Encode(message)
}

func (c *msgpackCodec) Unmarshal(binary []byte, message any) error {
	decoder := msgpack.NewDecoderBytes(binary, new(msgpack.MsgpackHandle))
	return decoder.Decode(message)
}

// readOnlyCodecs is a read-only interface to a map of named codecs.
type readOnlyCodecs interface {
	// Get gets the Codec with the given name.
	Get(string) Codec
	// Protobuf gets the user-supplied protobuf codec, falling back to the default
	// implementation if necessary.
	//
	// This is helpful in the gRPC protocol, where the wire protocol requires
	// marshaling protobuf structs to binary even if the RPC procedures were
	// generated from a different IDL.
	Protobuf() Codec
	// Names returns a copy of the registered codec names. The returned slice is
	// safe for the caller to mutate.
	Names() []string
}

func newReadOnlyCodecs(nameToCodec map[string]Codec) readOnlyCodecs {
	return &codecMap{
		nameToCodec: nameToCodec,
	}
}

type codecMap struct {
	nameToCodec map[string]Codec
}

func (m *codecMap) Get(name string) Codec {
	return m.nameToCodec[name]
}

func (m *codecMap) Protobuf() Codec {
	if pb, ok := m.nameToCodec[codecNameProto]; ok {
		return pb
	}
	return &protoBinaryCodec{}
}

func (m *codecMap) Names() []string {
	names := make([]string, 0, len(m.nameToCodec))
	for name := range m.nameToCodec {
		names = append(names, name)
	}
	return names
}

func errNotProto(message any) error {
	if _, ok := message.(protoiface.MessageV1); ok {
		return fmt.Errorf("%T uses github.com/golang/protobuf, but triple only supports google.golang.org/protobuf: see https://go.dev/blog/protobuf-apiv2", message)
	}
	return fmt.Errorf("%T doesn't implement proto.Message", message)
}

// Definitions from dubbogo/grpc-go
func getArgType(v any) string {
	if v == nil {
		return "V"
	}

	switch v := v.(type) {
	// Serialized tags for base types
	case bool:
		return "boolean"
	case []bool:
		return "[Z"
	case byte:
		return "byte"
	case []byte:
		return "[B"
	case int8:
		return "byte"
	case []int8:
		return "[B"
	case int16:
		return "short"
	case []int16:
		return "[S"
	case uint16: // Equivalent to Char of Java
		return "char"
	case []uint16:
		return "[C"
	case int: // Equivalent to Long of Java
		return "long"
	case []int:
		return "[J"
	case int32:
		return "int"
	case []int32:
		return "[I"
	case int64:
		return "long"
	case []int64:
		return "[J"
	case time.Time:
		return "java.util.Date"
	case []time.Time:
		return "[Ljava.util.Date"
	case float32:
		return "float"
	case []float32:
		return "[F"
	case float64:
		return "double"
	case []float64:
		return "[D"
	case string:
		return "java.lang.String"
	case []string:
		return "[Ljava.lang.String;"
	case []hessian.Object:
		return "[Ljava.lang.Object;"
	case map[any]any:
		// return  "java.util.HashMap"
		return "java.util.Map"
	case hessian.POJOEnum:
		return v.JavaClassName()
	//  Serialized tags for complex types
	default:
		t := reflect.TypeOf(v)
		if reflect.Pointer == t.Kind() {
			t = t.Elem()
		}
		switch t.Kind() {
		case reflect.Struct:
			v, ok := v.(hessian.POJO)
			if ok {
				return v.JavaClassName()
			}
			return "java.lang.Object"
		case reflect.Slice, reflect.Array:
			if t.Elem().Kind() == reflect.Struct {
				return "[Ljava.lang.Object;"
			}
			// return "java.util.ArrayList"
			return "java.util.List"
		case reflect.Map: // Enter here, map may be map[string]int
			return "java.util.Map"
		default:
			return ""
		}
	}
}

func reflectResponse(in any, out any) error {
	if in == nil {
		return perrors.Errorf("@in is nil")
	}

	if out == nil {
		return perrors.Errorf("@out is nil")
	}
	if reflect.TypeOf(out).Kind() != reflect.Pointer {
		return perrors.Errorf("@out should be a pointer")
	}

	inValue := hessian.EnsurePackValue(in)
	outValue := hessian.EnsurePackValue(out)

	outType := outValue.Type().String()
	if outType == "interface {}" || outType == "*interface {}" {
		hessian.SetValue(outValue, inValue)
		return nil
	}

	switch inValue.Type().Kind() {
	case reflect.Slice, reflect.Array:
		return copySlice(inValue, outValue)
	case reflect.Map:
		return copyMap(inValue, outValue)
	default:
		hessian.SetValue(outValue, inValue)
	}

	return nil
}

// copySlice copy from inSlice to outSlice
func copySlice(inSlice, outSlice reflect.Value) error {
	if inSlice.IsNil() {
		return perrors.New("@in is nil")
	}
	if inSlice.Kind() != reflect.Slice {
		return perrors.Errorf("@in is not slice, but %v", inSlice.Kind())
	}

	for outSlice.Kind() == reflect.Pointer {
		outSlice = outSlice.Elem()
	}

	size := inSlice.Len()
	outSlice.Set(reflect.MakeSlice(outSlice.Type(), size, size))

	for i := range size {
		inSliceValue := inSlice.Index(i)
		if !inSliceValue.Type().AssignableTo(outSlice.Index(i).Type()) {
			return perrors.Errorf("in element type [%s] can not assign to out element type [%s]",
				inSliceValue.Type().String(), outSlice.Type().String())
		}
		outSlice.Index(i).Set(inSliceValue)
	}

	return nil
}

// tripleServerCodecSession is a per-request Codec for the triple server that
// handles both IDL and Non-IDL formats.
//
// SerializeType is request-scoped state that the Codec interface
// (Marshal/Unmarshal) has no channel to surface. The session object IS that
// channel: Unmarshal captures SerializeType from the TripleRequestWrapper in a
// single decode, and Marshal reads it to wrap the response in a
// TripleResponseWrapper.
type tripleServerCodecSession struct {
	delegate             Codec  // IDL path codec, resolved from Content-Type
	serializeType        string // captured by Unmarshal when Non-IDL; read by Marshal
	allowedSerializeType string // provider "serialization" param; effective allowlist = {hessian2} ∪ {this}. TODO: support Java's multi-valued prefer-serialization
}

var _ Codec = (*tripleServerCodecSession)(nil)

func (s *tripleServerCodecSession) Name() string { return s.delegate.Name() }

// checkAllowed enforces the provider-side serialization allowlist.
// hessian2 is always allowed (Non-IDL interop default); any other name must
// match the provider's configured serialization.
func (s *tripleServerCodecSession) checkAllowed(codecName string) error {
	if codecName == codecNameHessian2 || codecName == s.allowedSerializeType {
		return nil
	}
	return fmt.Errorf("serialize type %q not allowed by provider (allowed: %s, %s)",
		codecName, codecNameHessian2, s.allowedSerializeType)
}

func (s *tripleServerCodecSession) Unmarshal(data []byte, message any) error {
	if _, isProto := message.(proto.Message); isProto {
		// IDL: standard proto message.
		return s.delegate.Unmarshal(data, message)
	}
	// Non-IDL: decode the TripleRequestWrapper once, capturing SerializeType
	// for the subsequent response Marshal and decoding the inner args in the
	// same pass.
	var reqWrapper interoperability.TripleRequestWrapper
	if err := proto.Unmarshal(data, &reqWrapper); err != nil {
		return fmt.Errorf("unmarshal triple wrapper request: %w", err)
	}
	s.serializeType = reqWrapper.SerializeType
	inner, err := resolveInnerCodec(reqWrapper.SerializeType)
	if err != nil {
		return fmt.Errorf("unmarshal triple wrapper request: %w", err)
	}
	if err := s.checkAllowed(inner.Name()); err != nil {
		return fmt.Errorf("unmarshal triple wrapper request: %w", err)
	}
	return unmarshalWrapperRequestArgs(&reqWrapper, inner, message)
}

func (s *tripleServerCodecSession) Marshal(message any) ([]byte, error) {
	if _, isProto := message.(proto.Message); isProto {
		// IDL: standard proto message.
		return s.delegate.Marshal(message)
	}
	// Non-IDL: wrap the response in a TripleResponseWrapper whose Data is
	// serialized with the inner codec resolved from the request's SerializeType.
	inner, err := resolveInnerCodec(s.serializeType)
	if err != nil {
		return nil, fmt.Errorf("marshal triple wrapper response: %w", err)
	}
	payload := message
	var isVoid bool
	if container, ok := message.([]any); ok {
		// The production handler packs exactly one return value as
		// []any{result} (server.go wrapTripleResponse). More elements indicate
		// a programming error; fail loudly instead of silently truncating.
		switch len(container) {
		case 0:
			isVoid = true
		case 1:
			payload = container[0]
			if payload == nil {
				isVoid = true
			}
		default:
			return nil, fmt.Errorf("marshal triple wrapper response: expected at most 1 return value, got %d", len(container))
		}
	}
	var data []byte
	if !isVoid {
		data, err = inner.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("marshal triple wrapper response data: %w", err)
		}
	}
	// Use inner.Name() instead of s.serializeType so that an absent SerializeType
	// (defaulted to hessian2 by resolveInnerCodec) is normalized on the wire.
	return proto.Marshal(&interoperability.TripleResponseWrapper{
		SerializeType: inner.Name(),
		Data:          data,
	})
}

// unmarshalWrapperRequestArgs decodes the inner args of a TripleRequestWrapper
// into message. message may be []any (multi-arg generic call) or a single
// value (single-arg call packed as a one-element wrapper).
func unmarshalWrapperRequestArgs(w *interoperability.TripleRequestWrapper, inner Codec, message any) error {
	if params, isSlice := message.([]any); isSlice {
		if len(w.Args) != len(params) {
			return fmt.Errorf("triple wrapper request: expected %d params, got %d args", len(params), len(w.Args))
		}
		for i, arg := range w.Args {
			if err := inner.Unmarshal(arg, params[i]); err != nil {
				return fmt.Errorf("triple wrapper request arg[%d]: %w", i, err)
			}
		}
		return nil
	}
	// Single-arg call: the wrapper carries one arg decoded into message.
	if len(w.Args) != 1 {
		return fmt.Errorf("triple wrapper request: expected 1 arg for single param, got %d", len(w.Args))
	}
	return inner.Unmarshal(w.Args[0], message)
}

// copyMap copy from in map to out map
func copyMap(inMapValue, outMapValue reflect.Value) error {
	if inMapValue.IsNil() {
		return perrors.New("@in is nil")
	}
	if !inMapValue.CanInterface() {
		return perrors.New("@in's Interface can not be used.")
	}
	if inMapValue.Kind() != reflect.Map {
		return perrors.Errorf("@in is not map, but %v", inMapValue.Kind())
	}

	outMapType := hessian.UnpackPtrType(outMapValue.Type())
	hessian.SetValue(outMapValue, reflect.MakeMap(outMapType))

	outKeyType := outMapType.Key()

	outMapValue = hessian.UnpackPtrValue(outMapValue)
	outValueType := outMapValue.Type().Elem()

	for _, inKey := range inMapValue.MapKeys() {
		inValue := inMapValue.MapIndex(inKey)

		if !inKey.Type().AssignableTo(outKeyType) {
			return perrors.Errorf("in Key:{type:%s, value:%#v} can not assign to out Key:{type:%s} ",
				inKey.Type().String(), inKey, outKeyType.String())
		}
		if !inValue.Type().AssignableTo(outValueType) {
			return perrors.Errorf("in Value:{type:%s, value:%#v} can not assign to out value:{type:%s}",
				inValue.Type().String(), inValue, outValueType.String())
		}
		outMapValue.SetMapIndex(inKey, inValue)
	}

	return nil
}
