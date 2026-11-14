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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/proto"

	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/interoperability"
)

type runtimeJSON struct{}

func (runtimeJSON) Name() string { return "private-json" }
func (runtimeJSON) Marshal(v any) ([]byte, error) {
	b, e := json.Marshal(v)
	return append([]byte("PRIVATE:"), b...), e
}
func (runtimeJSON) Unmarshal(b []byte, v any) error {
	if !bytes.HasPrefix(b, []byte("PRIVATE:")) {
		return fmt.Errorf("private prefix missing")
	}
	return json.Unmarshal(b[8:], v)
}
func TestConfiguredSerializationSessions(t *testing.T) {
	inner := runtimeJSON{}
	client := &protoWrapperCodec{innerCodec: inner, runtimeCodecs: map[string]Codec{inner.Name(): inner}}
	session := &tripleServerCodecSession{delegate: &protoBinaryCodec{}, runtimeCodecs: map[string]Codec{inner.Name(): inner}, messageCodecs: map[string]Codec{}}
	request, err := client.Marshal([]any{"hello"})
	require.NoError(t, err)
	var argument string
	require.NoError(t, session.Unmarshal(request, []any{&argument}))
	require.Equal(t, "hello", argument)
	response, err := session.Marshal([]any{"world"})
	require.NoError(t, err)
	var reply string
	require.NoError(t, client.Unmarshal(response, &reply))
	require.Equal(t, "world", reply)
	other, err := proto.Marshal(&interoperability.TripleResponseWrapper{SerializeType: "hessian2"})
	require.NoError(t, err)
	require.ErrorContains(t, client.Unmarshal(other, &reply), "disabled")
	malformed, err := proto.Marshal(&interoperability.TripleRequestWrapper{SerializeType: inner.Name(), Args: [][]byte{[]byte("invalid")}})
	require.NoError(t, err)
	require.ErrorContains(t, session.Unmarshal(malformed, []any{&argument}), "prefix missing")
	// Legacy gRPC-wire layout is recognized by fields, while payload errors never
	// trigger a different codec. This fixture schema is emitted by baseline's
	// protoWrapperCodec.Marshal for responses on the gRPC fallback path.
	legacy, err := newProtoWrapperCodec(inner).Marshal([]any{"legacy"})
	require.NoError(t, err)
	require.NoError(t, client.Unmarshal(legacy, &reply))
	require.Equal(t, "legacy", reply)
	response, err = session.Marshal([]any{nil})
	require.NoError(t, err)
	require.NoError(t, client.Unmarshal(response, &reply))
	_, err = client.Marshal(SerializationArguments{Values: []any{"hello"}, TypeNames: []string{"java.lang.String", "extra"}})
	require.ErrorContains(t, err, "declared type count")
	_, err = client.Marshal([]any{struct{ Value string }{"a"}})
	require.ErrorContains(t, err, "ArgumentTypeResolver")
}

func TestDisabledProtobufEmptyEnvelope(t *testing.T) {
	for _, wire := range []string{"grpc", "triple"} {
		for _, fast := range []bool{true, false} {
			for _, allowed := range []bool{true, false} {
				t.Run(fmt.Sprintf("%s/fast=%t/allowed=%t", wire, fast, allowed), func(t *testing.T) {
					builtin := BuiltinSerializationCodecs()["proto"]
					messages := map[string]Codec{}
					if allowed {
						messages["proto"] = builtin
					}
					_, opts := ConfiguredSerializationOptions(builtin, map[string]Codec{}, messages, false, false)
					var calls atomic.Int64
					mux := http.NewServeMux()
					mux.Handle("/review.Service/Run", NewUnaryHandler("/review.Service/Run", func() any { return &wrapperspb.StringValue{} }, func(context.Context, *Request) (*Response, error) {
						calls.Add(1)
						return NewResponse(&emptypb.Empty{}), nil
					}, opts...))
					srv := httptest.NewUnstartedServer(mux)
					srv.EnableHTTP2 = true
					srv.StartTLS()
					defer srv.Close()
					var clientOpts []ClientOption
					if wire == "triple" {
						clientOpts = append(clientOpts, WithTriple())
					}
					if !fast {
						clientOpts = append(clientOpts, WithoutUnaryFastPath())
					}
					client := NewClient(srv.Client(), srv.URL+"/review.Service", clientOpts...)
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					for _, input := range []*wrapperspb.StringValue{{}, wrapperspb.String("value")} {
						err := client.CallUnary(ctx, NewRequest(input), "Run", NewResponse(&emptypb.Empty{}))
						if allowed {
							require.NoError(t, err)
						} else {
							require.ErrorContains(t, err, "disabled")
						}
					}
					if allowed {
						require.EqualValues(t, 2, calls.Load())
					} else {
						require.Zero(t, calls.Load(), "disabled messages must not execute business code")
					}
				})
			}
		}
	}
}

type codecResolverProbe struct {
	runtimeJSON
	calls    int
	resolved string
	err      error
}

func (c *codecResolverProbe) ArgumentType(any) (string, error) { c.calls++; return c.resolved, c.err }
func TestCodecDeclaredTypesAndResolver(t *testing.T) {
	sentinel := errors.New("resolver error")
	c := &codecResolverProbe{resolved: "resolved.Type"}
	wrapper := &protoWrapperCodec{innerCodec: c, runtimeCodecs: map[string]Codec{c.Name(): c}}
	for _, test := range []struct {
		name     string
		declared []string
		resolved string
		err      error
		want     string
		bad      bool
	}{
		{"declaration-wins", []string{"declared.Type"}, "ignored", sentinel, "declared.Type", false},
		{"resolver", nil, "resolved.Type", nil, "resolved.Type", false},
		{"resolver-error", nil, "", sentinel, "", true},
		{"resolver-empty", nil, "", nil, "", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			c.resolved, c.err = test.resolved, test.err
			before := c.calls
			b, err := wrapper.Marshal(SerializationArguments{Values: []any{struct{ Value string }{"x"}}, TypeNames: test.declared})
			if test.bad {
				require.Error(t, err)
				if test.err != nil {
					require.ErrorIs(t, err, sentinel)
				}
			} else {
				require.NoError(t, err)
				var message interoperability.TripleRequestWrapper
				require.NoError(t, proto.Unmarshal(b, &message))
				require.Equal(t, []string{test.want}, message.ArgTypes)
			}
			if len(test.declared) > 0 {
				require.Equal(t, before, c.calls)
			} else {
				require.Equal(t, before+1, c.calls)
			}
		})
	}
}

type codecDecodeProbe struct {
	name  string
	calls int
	err   error
}

func (c *codecDecodeProbe) Name() string                { return c.name }
func (c *codecDecodeProbe) Marshal(any) ([]byte, error) { return []byte("payload"), nil }
func (c *codecDecodeProbe) Unmarshal([]byte, any) error { c.calls++; return c.err }
func TestCodecRuntimeMapIsolationAndRejection(t *testing.T) {
	selected := runtimeJSON{}
	other := &codecDecodeProbe{name: "other"}
	inner := map[string]Codec{selected.Name(): selected}
	messages := map[string]Codec{"proto": &protoBinaryCodec{}}
	clientOpts, handlerOpts := ConfiguredSerializationOptions(selected, inner, messages, true, true)
	delete(inner, selected.Name())
	inner["hessian2"] = other
	delete(messages, "proto")
	messages["other"] = other
	clientConfig, err := newClientConfig("http://127.0.0.1/test.Service", clientOpts)
	require.Nil(t, err)
	client := clientConfig.Codec.(*protoWrapperCodec)
	handler := newHandlerConfig("/test.Service/Echo", handlerOpts)
	require.Contains(t, handler.InnerCodecs, selected.Name())
	require.NotContains(t, handler.InnerCodecs, "hessian2")
	require.Contains(t, handler.MessageCodecs, "proto")
	require.NotContains(t, handler.MessageCodecs, "other")
	for _, name := range []string{"", "hessian4", "unknown"} {
		body, err := proto.Marshal(&interoperability.TripleResponseWrapper{SerializeType: name, Data: []byte("payload")})
		require.NoError(t, err)
		require.Error(t, client.Unmarshal(body, new(string)))
		session := &tripleServerCodecSession{delegate: &protoBinaryCodec{}, runtimeCodecs: handler.InnerCodecs, messageCodecs: handler.MessageCodecs}
		body, err = proto.Marshal(&interoperability.TripleRequestWrapper{SerializeType: name, Args: [][]byte{[]byte("payload")}})
		require.NoError(t, err)
		require.Error(t, session.Unmarshal(body, []any{new(string)}))
	}
	require.Zero(t, other.calls)
}
func TestCodecResponseSchemasAndPayloadErrors(t *testing.T) {
	sentinel := errors.New("bad inner payload")
	allowed := &codecDecodeProbe{name: "allowed", err: sentinel}
	other := &codecDecodeProbe{name: "other"}
	c := &protoWrapperCodec{innerCodec: allowed, runtimeCodecs: map[string]Codec{"allowed": allowed, "other": other}}
	for _, test := range []struct {
		name    string
		wire    proto.Message
		raw     []byte
		payload bool
	}{
		{name: "invalid-schema", raw: []byte{255}},
		{name: "new-disabled", wire: &interoperability.TripleResponseWrapper{SerializeType: "disabled", Data: []byte("payload")}},
		{name: "legacy-disabled", wire: &interoperability.TripleRequestWrapper{SerializeType: "disabled", Args: [][]byte{[]byte("payload")}}},
		{name: "legacy-many", wire: &interoperability.TripleRequestWrapper{SerializeType: "allowed", Args: [][]byte{[]byte("a"), []byte("b")}}},
		{name: "new-payload", wire: &interoperability.TripleResponseWrapper{SerializeType: "allowed", Data: []byte("bad")}, payload: true},
		{name: "legacy-payload", wire: &interoperability.TripleRequestWrapper{SerializeType: "allowed", Args: [][]byte{[]byte("bad")}}, payload: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := allowed.calls
			body := test.raw
			if test.wire != nil {
				var err error
				body, err = proto.Marshal(test.wire)
				require.NoError(t, err)
			}
			err := c.Unmarshal(body, new(string))
			require.Error(t, err)
			if test.payload {
				require.ErrorIs(t, err, sentinel)
				require.Equal(t, before+1, allowed.calls)
			} else {
				require.Equal(t, before, allowed.calls)
			}
			require.Zero(t, other.calls)
		})
	}
}

type codecMessageProbe struct {
	name    string
	err     error
	appends atomic.Int64
}

func (c *codecMessageProbe) Name() string { return c.name }
func (c *codecMessageProbe) Marshal(v any) ([]byte, error) {
	if c.err != nil {
		return nil, c.err
	}
	return json.Marshal(v)
}
func (c *codecMessageProbe) Unmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }

type codecAppenderProbe struct{ *codecMessageProbe }

func (c codecAppenderProbe) MarshalAppend(dst []byte, v any) ([]byte, error) {
	c.appends.Add(1)
	b, err := c.Marshal(v)
	if err != nil {
		return nil, err
	}
	return append(dst, b...), nil
}
func TestCodecMessageSessionAppender(t *testing.T) {
	sentinel := errors.New("marshal failed")
	for _, appender := range []bool{false, true} {
		t.Run(fmt.Sprint(appender), func(t *testing.T) {
			base := &codecMessageProbe{name: "codec-message"}
			var delegate Codec = base
			if appender {
				delegate = codecAppenderProbe{base}
			}
			session := &tripleServerCodecSession{delegate: delegate, runtimeCodecs: map[string]Codec{}, messageCodecs: map[string]Codec{delegate.Name(): delegate}}
			message := map[string]string{"value": "hello"}
			expected, err := session.Marshal(message)
			require.NoError(t, err)
			got, err := session.MarshalAppend([]byte("PREFIX:"), message)
			require.NoError(t, err)
			require.Equal(t, append([]byte("PREFIX:"), expected...), got)
			base.err = sentinel
			_, err = session.MarshalAppend([]byte("PREFIX:"), message)
			require.ErrorIs(t, err, sentinel)
			if appender {
				require.EqualValues(t, 2, base.appends.Load())
			}
		})
	}
	// Configured handler path, rather than directly invoking a session helper.
	probe := &codecMessageProbe{name: "codec-message-http"}
	delegate := codecAppenderProbe{probe}
	clientOpts, opts := ConfiguredSerializationOptions(delegate, map[string]Codec{}, map[string]Codec{delegate.Name(): delegate}, false, true)
	handler := NewUnaryHandler("/codec.Service/Echo", func() any { return new(string) }, func(context.Context, *Request) (*Response, error) { return NewResponse("ok"), nil }, opts...)
	server := httptest.NewServer(handler)
	defer server.Close()
	client := NewClient(server.Client(), server.URL+"/codec.Service", clientOpts...)
	require.NoError(t, client.CallUnary(context.Background(), NewRequest("hello"), "Echo", NewResponse(new(string))))
	require.Positive(t, probe.appends.Load())
}
func TestCodecStreamingEmptyMessagePolicy(t *testing.T) {
	for _, allowed := range []bool{false, true} {
		t.Run(fmt.Sprint(allowed), func(t *testing.T) {
			messages := map[string]Codec{}
			if allowed {
				messages["proto"] = &protoBinaryCodec{}
			}
			_, opts := ConfiguredSerializationOptions(&protoBinaryCodec{}, map[string]Codec{}, messages, false, false)
			var calls atomic.Int64
			handler := NewClientStreamHandler("/codec.Service/Stream", func(_ context.Context, stream *ClientStream) (*Response, error) {
				if !stream.Receive(&emptypb.Empty{}) {
					return nil, stream.Err()
				}
				calls.Add(1)
				return NewResponse(&emptypb.Empty{}), nil
			}, opts...)
			server := httptest.NewUnstartedServer(handler)
			server.EnableHTTP2 = true
			server.StartTLS()
			defer server.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			client := NewClient(server.Client(), server.URL+"/codec.Service")
			raw, err := client.CallClientStream(ctx, "Stream")
			require.NoError(t, err)
			stream := raw
			_ = stream.Send(&emptypb.Empty{})
			err = stream.CloseAndReceive(NewResponse(&emptypb.Empty{}))
			if allowed {
				require.NoError(t, err)
				require.EqualValues(t, 1, calls.Load())
			} else {
				require.ErrorContains(t, err, "disabled")
				require.Zero(t, calls.Load())
			}
		})
	}
}
