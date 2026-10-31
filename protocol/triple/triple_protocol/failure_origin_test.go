/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements. See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License. You may obtain a copy of the License at
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
	"errors"
	"net"
	"net/http"
	"testing"
	"time"
)

import (
	"google.golang.org/protobuf/types/known/wrapperspb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
)

func TestCodecFailureOrigins(t *testing.T) {
	for _, codec := range []Codec{errorCodec{}, noAppenderCodec{errorCodec{}}} {
		w := envelopeWriter{codec: codec, bufferPool: newBufferPool(), writer: &bytes.Buffer{}}
		err := w.Marshal(struct{}{})
		if err == nil || CodeOf(err) != CodeInternal || rpcerror.Of(err) != rpcerror.Codec {
			t.Fatalf("missing codec origin: %v", err)
		}
		var original *Error
		if !errors.As(err, &original) || original != err || IsWireError(err) {
			t.Fatal("changed outer Triple error")
		}
		// A fallback success must not retain the primary codec's failure.
		w.backupCodec = &protoJSONCodec{}
		if err := w.Marshal(wrapperspb.String("fallback")); err != nil {
			t.Fatal(err)
		}
	}
	cause := errors.New("marshal fixture")
	err := errorf(CodeInternal, "marshal message: %w", rpcerror.Mark(cause, rpcerror.Codec))
	if !errors.Is(err, cause) || err.Error() != "internal: marshal message: marshal fixture" {
		t.Fatalf("changed cause or text: %v", err)
	}
}

func TestNetworkOriginThroughUnaryTransports(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	_ = listener.Close()
	for _, fast := range []bool{true, false} {
		opts := []ClientOption{WithTriple(), WithTimeout(time.Second)}
		if !fast {
			opts = append(opts, WithoutUnaryFastPath())
		}
		client := NewClient(&http.Client{Timeout: time.Second}, "http://"+address+"/fixture.Service", opts...)
		err := client.CallUnary(context.Background(), NewRequest(wrapperspb.String("request")), "Call", NewResponse(new(wrapperspb.StringValue)))
		var cause *net.OpError
		if !errors.As(err, &cause) || CodeOf(err) != CodeUnavailable || rpcerror.Of(err) != rpcerror.Network {
			t.Fatalf("fast=%v lost source/cause/code: %v", fast, err)
		}
	}
}
