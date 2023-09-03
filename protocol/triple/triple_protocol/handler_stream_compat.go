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
	"github.com/dubbogo/grpc-go"
	"github.com/dubbogo/grpc-go/metadata"
)

type compatHandlerStream struct {
	ctx  context.Context
	conn StreamingHandlerConn
}

func (c *compatHandlerStream) SetHeader(md metadata.MD) error {
	// todo(DMwangnima): add header method for streaming
	return nil
}

func (c *compatHandlerStream) SendHeader(md metadata.MD) error {
	// todo(DMwangnima): add header method for streaming
	return nil
}

func (c *compatHandlerStream) SetTrailer(md metadata.MD) {
	// todo(DMwangnima): add trailer method for streaming
	return
}

func (c *compatHandlerStream) Context() context.Context {
	return c.ctx
}

func (c *compatHandlerStream) SendMsg(m interface{}) error {
	return c.conn.Send(m)
}

func (c *compatHandlerStream) RecvMsg(m interface{}) error {
	return c.conn.Receive(m)
}

func NewCompatStreamHandler(
	procedure string,
	srv interface{},
	typ StreamType,
	implementation func(srv interface{}, stream grpc.ServerStream) error,
	options ...HandlerOption,
) *Handler {
	return newStreamHandler(
		procedure,
		typ,
		func(ctx context.Context, conn StreamingHandlerConn) error {
			stream := &compatHandlerStream{
				ctx:  ctx,
				conn: conn,
			}
			return implementation(srv, stream)
		},
		options...,
	)
}
