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
	"context"
	"net"
	"net/http"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestLegacyWrappedClientCallsConfiguredHandler(t *testing.T) {
	for _, format := range []string{"hessian2", "msgpack"} {
		for _, allowed := range []bool{true, false} {
			t.Run(fmtLegacyCase(format, allowed), func(t *testing.T) {
				u := common.NewURLWithOptions(common.WithProtocol("tri"), common.WithPath("codec.LegacyWire"), common.WithParamsValue("interface", "codec.LegacyWire"), common.WithParamsValue("serialization", format))
				if !allowed {
					u.SetParam("accepted-serializations", "proto")
				}
				_, options, _, err := SerializationOptions(u, "provider")
				require.NoError(t, err)
				defer func() { require.NoError(t, (&TripleProtocol{}).ReleaseCodec(u, "provider")) }()
				var called atomic.Int64
				handler := tri.NewUnaryHandler("/codec.LegacyWire/Echo", func() any { return []any{new(string)} }, func(_ context.Context, req *tri.Request) (*tri.Response, error) {
					called.Add(1)
					value := *(req.Any().([]any)[0].(*string))
					return tri.NewResponse("hello " + value), nil
				}, options...)
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				require.NoError(t, err)
				server := &http.Server{Handler: h2c.NewHandler(handler, &http2.Server{})}
				defer server.Close()
				go func() { _ = server.Serve(listener) }()
				ref, err := common.NewURL("tri://"+listener.Addr().String()+"/codec.LegacyWire", common.WithParamsValue("interface", "codec.LegacyWire"), common.WithParamsValue("serialization", format), common.WithParamsValue("timeout", "3s"))
				require.NoError(t, err)
				client, err := NewDubbo3Invoker(ref)
				require.NoError(t, err)
				defer client.Destroy()
				var reply string
				inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("Echo"), invocation.WithParameterValues([]reflect.Value{reflect.ValueOf("world")}), invocation.WithReply(&reply))
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				result := client.Invoke(ctx, inv)
				if allowed {
					require.NoError(t, result.Error())
					require.Equal(t, "hello world", reply)
					require.EqualValues(t, 1, called.Load())
				} else {
					require.ErrorContains(t, result.Error(), "disabled")
					require.Zero(t, called.Load())
				}
			})
		}
	}
}
func fmtLegacyCase(format string, allowed bool) string {
	if allowed {
		return format + "/allowed"
	}
	return format + "/denied"
}
