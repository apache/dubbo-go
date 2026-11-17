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
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/emptypb"
)

type emptyClientCodec struct {
	calls  atomic.Int64
	reject bool
}

func (*emptyClientCodec) Name() string                { return "empty-test" }
func (*emptyClientCodec) Marshal(any) ([]byte, error) { return []byte("request"), nil }
func (c *emptyClientCodec) Unmarshal(data []byte, _ any) error {
	c.calls.Add(1)
	if len(data) == 0 && c.reject {
		return errors.New("plugin rejects empty payload")
	}
	return nil
}
func TestConfiguredClientEmptyResponses(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
		w.Header().Set("Trailer", "Grpc-Status")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(make([]byte, 5))
		w.Header().Set("Grpc-Status", "0")
	}))
	defer srv.Close()
	for _, kind := range []string{"plugin-reject", "plugin-accept", "proto", "wrapped-allow", "wrapped-deny", "legacy"} {
		for _, fast := range []bool{false, true} {
			t.Run(kind+"/fast="+map[bool]string{true: "true", false: "false"}[fast], func(t *testing.T) {
				plugin := &emptyClientCodec{reject: kind == "plugin-reject" || kind == "legacy"}
				var selected Codec = plugin
				var input any = &emptypb.Empty{}
				var reply any = &emptypb.Empty{}
				wrapped := kind == "wrapped-allow" || kind == "wrapped-deny"
				inner := map[string]Codec{}
				if kind == "proto" {
					selected = BuiltinSerializationCodecs()["proto"]
				}
				if wrapped {
					selected = BuiltinSerializationCodecs()["hessian2"]
					input = []any{"request"}
					reply = new(string)
					if kind == "wrapped-allow" {
						inner["hessian2"] = selected
					}
				}
				opts, _ := ConfiguredSerializationOptions(selected, inner, map[string]Codec{selected.Name(): selected}, wrapped, true)
				if kind == "legacy" {
					opts = []ClientOption{WithCodec(selected)}
				}
				if !fast {
					opts = append(opts, WithoutUnaryFastPath())
				}
				cli := NewClient(srv.Client(), srv.URL+"/codec.Empty", opts...)
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				err := cli.CallUnary(ctx, NewRequest(input), "Call", NewResponse(reply))
				rejected := kind == "plugin-reject" || kind == "wrapped-deny"
				if rejected {
					require.Error(t, err)
				} else {
					require.NoError(t, err)
				}
				stream, err := cli.CallServerStream(ctx, NewRequest(input), "Call")
				require.NoError(t, err)
				require.Equal(t, !rejected, stream.Receive(reply))
				if rejected {
					require.Error(t, stream.Err())
				} else {
					require.NoError(t, stream.Err())
				}
				require.NoError(t, stream.Close())
				expected := int64(0)
				if kind == "plugin-reject" || kind == "plugin-accept" {
					expected = 2
				}
				require.Equal(t, expected, plugin.calls.Load())
			})
		}
	}
}
