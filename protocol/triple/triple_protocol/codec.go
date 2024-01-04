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
	"encoding/json"
	"errors"
	"fmt"
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/dubbogo/grpc-go/encoding"
	"github.com/dubbogo/grpc-go/encoding/proto_wrapper_api"
	"github.com/dubbogo/grpc-go/encoding/tools"
)

import (
	"google.golang.org/protobuf/encoding/protojson"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/runtime/protoiface"

	msgpack "github.com/ugorji/go/codec"
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
	Marshal(interface{}) ([]byte, error)
	// Unmarshal unmarshals the given message.
	//
	// Unmarshal may expect a specific type of message, and will error if this
	// type is not given.
	Unmarshal([]byte, interface{}) error
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
	MarshalStable(interface{}) ([]byte, error)

	// IsBinary returns true if the marshaled data is binary for this codec.
	//
	// If this function returns false, the data returned from Marshal and
	// MarshalStable are considered valid text and may be used in contexts
	// where text is expected.
	IsBinary() bool
}

type protoBinaryCodec struct{}

var _ Codec = (*protoBinaryCodec)(nil)

func (c *protoBinaryCodec) Name() string { return codecNameProto }

func (c *protoBinaryCodec) Marshal(message interface{}) ([]byte, error) {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return nil, errNotProto(message)
	}
	return proto.Marshal(protoMessage)
}

func (c *protoBinaryCodec) Unmarshal(data []byte, message interface{}) error {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return errNotProto(message)
	}
	return proto.Unmarshal(data, protoMessage)
}

func (c *protoBinaryCodec) MarshalStable(message interface{}) ([]byte, error) {
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

func (c *protoJSONCodec) Marshal(message interface{}) ([]byte, error) {
	protoMessage, ok := message.(proto.Message)
	if !ok {
		return nil, errNotProto(message)
	}
	var options protojson.MarshalOptions
	return options.Marshal(protoMessage)
}

func (c *protoJSONCodec) Unmarshal(binary []byte, message interface{}) error {
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

func (c *protoJSONCodec) MarshalStable(message interface{}) ([]byte, error) {
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

// todo(DMwangnima): add unit tests
type protoWrapperCodec struct {
	innerCodec Codec
}

func (c *protoWrapperCodec) Name() string {
	return c.innerCodec.Name()
}

func (c *protoWrapperCodec) Marshal(message interface{}) ([]byte, error) {
	reqs, ok := message.([]interface{})
	if !ok {
		return c.innerCodec.Marshal(message)
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
		reqsTypes[i] = encoding.GetArgType(req)
	}

	wrapperReq := &proto_wrapper_api.TripleRequestWrapper{
		SerializeType: c.innerCodec.Name(),
		Args:          reqsBytes,
		ArgTypes:      reqsTypes,
	}

	return proto.Marshal(wrapperReq)
}

func (c *protoWrapperCodec) Unmarshal(binary []byte, message interface{}) error {
	params, ok := message.([]interface{})
	if !ok {
		return c.innerCodec.Unmarshal(binary, message)
	}

	var wrapperReq proto_wrapper_api.TripleRequestWrapper
	if err := proto.Unmarshal(binary, &wrapperReq); err != nil {
		return err
	}
	if len(wrapperReq.Args) != len(params) {
		return fmt.Errorf("error, request params len is %d, but has %d actually", len(wrapperReq.Args), len(params))
	}

	for i, arg := range wrapperReq.Args {
		if err := c.innerCodec.Unmarshal(arg, params[i]); err != nil {
			return err
		}
	}

	return nil
}

func newProtoWrapperCodec(innerCodec Codec) *protoWrapperCodec {
	return &protoWrapperCodec{innerCodec: innerCodec}
}

// todo(DMwangnima): add unit tests
type hessian2Codec struct{}

func (h *hessian2Codec) Name() string {
	return codecNameHessian2
}

func (c *hessian2Codec) Marshal(message interface{}) ([]byte, error) {
	encoder := hessian.NewEncoder()
	if err := encoder.Encode(message); err != nil {
		return nil, err
	}

	return encoder.Buffer(), nil
}

func (c *hessian2Codec) Unmarshal(binary []byte, message interface{}) error {
	decoder := hessian.NewDecoder(binary)
	val, err := decoder.Decode()
	if err != nil {
		return err
	}
	return tools.ReflectResponse(val, message)
}

// todo(DMwangnima): add unit tests
type msgpackCodec struct{}

func (c *msgpackCodec) Name() string {
	return codecNameMsgPack
}

func (c *msgpackCodec) Marshal(message interface{}) ([]byte, error) {
	var out []byte
	encoder := msgpack.NewEncoderBytes(&out, new(msgpack.MsgpackHandle))
	return out, encoder.Encode(message)
}

func (c *msgpackCodec) Unmarshal(binary []byte, message interface{}) error {
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

func errNotProto(message interface{}) error {
	if _, ok := message.(protoiface.MessageV1); ok {
		return fmt.Errorf("%T uses github.com/golang/protobuf, but triple only supports google.golang.org/protobuf: see https://go.dev/blog/protobuf-apiv2", message)
	}
	return fmt.Errorf("%T doesn't implement proto.Message", message)
}
