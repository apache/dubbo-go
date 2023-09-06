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
	dubbo_protocol "dubbo.apache.org/dubbo-go/v3/protocol"
	"fmt"
	"github.com/dubbogo/grpc-go"
	"net/http"
)

type MethodHandler func(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error)

type tripleCompatInterceptor struct {
	spec        Spec
	peer        Peer
	header      http.Header
	procedure   string
	interceptor Interceptor
}

// be compatible with old triple-gen code
func (t *tripleCompatInterceptor) compatUnaryServerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	request := NewRequest(req)
	request.spec = t.spec
	request.peer = t.peer
	request.header = t.header

	unaryFunc := func(ctx context.Context, request AnyRequest) (AnyResponse, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		typed, ok := request.(*Request)
		if !ok {
			return nil, errorf(CodeInternal, "unexpected handler request type %T", request)
		}
		respRaw, err := handler(ctx, typed.Any())
		if respRaw == nil && err == nil {
			// This is going to panic during serialization. Debugging is much easier
			// if we panic here instead, so we can include the procedure name.
			panic(fmt.Sprintf("%s returned nil resp and nil error", t.procedure)) //nolint: forbidigo
		}
		resp, ok := respRaw.(*dubbo_protocol.RPCResult)
		if !ok {
			panic(fmt.Sprintf("%+v is not of type *RPCResult", respRaw))
		}
		// todo(DMwangnima): expose API for users to write response headers and trailers
		return NewResponse(resp.Rest), err
	}

	if t.interceptor != nil {
		unaryFunc = t.interceptor.WrapUnaryHandler(unaryFunc)
	}

	return unaryFunc(ctx, request)
}

func NewCompatUnaryHandler(
	procedure string,
	srv interface{},
	unary MethodHandler,
	options ...HandlerOption,
) *Handler {
	config := newHandlerConfig(procedure, options)

	implementation := func(ctx context.Context, conn StreamingHandlerConn) error {
		compatInterceptor := &tripleCompatInterceptor{
			spec:        conn.Spec(),
			peer:        conn.Peer(),
			header:      conn.RequestHeader(),
			procedure:   config.Procedure,
			interceptor: config.Interceptor,
		}
		decodeFunc := func(req interface{}) error {
			if err := conn.Receive(req); err != nil {
				return err
			}
			return nil
		}
		ctx = context.WithValue(ctx, "XXX_TRIPLE_GO_INTERFACE_NAME", config.Procedure)
		respRaw, err := unary(srv, ctx, decodeFunc, compatInterceptor.compatUnaryServerInterceptor)
		if err != nil {
			return err
		}
		resp := respRaw.(*Response)
		// merge headers
		mergeHeaders(conn.ResponseHeader(), resp.Header())
		mergeHeaders(conn.ResponseTrailer(), resp.Trailer())
		return conn.Send(resp.Any())
	}

	protocolHandlers := config.newProtocolHandlers(StreamTypeUnary)
	return &Handler{
		spec:             config.newSpec(StreamTypeUnary),
		implementation:   implementation,
		protocolHandlers: protocolHandlers,
		allowMethod:      sortedAllowMethodValue(protocolHandlers),
		acceptPost:       sortedAcceptPostValue(protocolHandlers),
	}
}
