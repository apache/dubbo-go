/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package benchmark

import (
	"context"
	"net/http"

	"dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
)

const (
	BenchmarkServiceName = "benchmark.BenchmarkService"
)

const (
	BenchmarkServiceUnaryCallProcedure  = "/benchmark.BenchmarkService/UnaryCall"
	BenchmarkServiceStreamCallProcedure = "/benchmark.BenchmarkService/StreamCall"
)

type TripleBenchmarkService interface {
	UnaryCall(ctx context.Context, req *BenchmarkRequest, opts ...client.CallOption) (*BenchmarkResponse, error)
	StreamCall(ctx context.Context, opts ...client.CallOption) (TripleBenchmarkService_StreamCallClient, error)
}

func NewTripleBenchmarkService(cli *client.Client, opts ...client.ReferenceOption) (TripleBenchmarkService, error) {
	conn, err := cli.DialWithInfo("benchmark.BenchmarkService", &TripleBenchmarkService_ClientInfo, opts...)
	if err != nil {
		return nil, err
	}
	return &TripleBenchmarkServiceImpl{
		conn: conn,
	}, nil
}

func SetTripleConsumerService(srv common.RPCService) {
	dubbo.SetConsumerServiceWithInfo(srv, &TripleBenchmarkService_ClientInfo)
}

type TripleBenchmarkServiceImpl struct {
	conn *client.Connection
}

func (c *TripleBenchmarkServiceImpl) UnaryCall(ctx context.Context, req *BenchmarkRequest, opts ...client.CallOption) (*BenchmarkResponse, error) {
	resp := new(BenchmarkResponse)
	if err := c.conn.CallUnary(ctx, []any{req}, resp, "UnaryCall", opts...); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *TripleBenchmarkServiceImpl) StreamCall(ctx context.Context, opts ...client.CallOption) (TripleBenchmarkService_StreamCallClient, error) {
	stream, err := c.conn.CallBidiStream(ctx, "StreamCall", opts...)
	if err != nil {
		return nil, err
	}
	rawStream := stream.(*triple_protocol.BidiStreamForClient)
	return &TripleBenchmarkServiceStreamCallClient{BidiStreamForClient: rawStream}, nil
}

type TripleBenchmarkService_StreamCallClient interface {
	Send(*BenchmarkRequest) error
	Recv() bool
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	Msg() *BenchmarkResponse
	Err() error
	Conn() (triple_protocol.StreamingClientConn, error)
	Close() error
}

type TripleBenchmarkServiceStreamCallClient struct {
	*triple_protocol.BidiStreamForClient
	msg *BenchmarkResponse
	err error
}

func (cli *TripleBenchmarkServiceStreamCallClient) Send(msg *BenchmarkRequest) error {
	return cli.BidiStreamForClient.Send(msg)
}

func (cli *TripleBenchmarkServiceStreamCallClient) Recv() bool {
	cli.msg = new(BenchmarkResponse)
	cli.err = cli.BidiStreamForClient.Receive(cli.msg)
	return cli.err == nil
}

func (cli *TripleBenchmarkServiceStreamCallClient) Msg() *BenchmarkResponse {
	return cli.msg
}

func (cli *TripleBenchmarkServiceStreamCallClient) Err() error {
	return cli.err
}

func (cli *TripleBenchmarkServiceStreamCallClient) Conn() (triple_protocol.StreamingClientConn, error) {
	return cli.BidiStreamForClient.Conn()
}

func (cli *TripleBenchmarkServiceStreamCallClient) Close() error {
	return cli.BidiStreamForClient.CloseResponse()
}

var TripleBenchmarkService_ClientInfo = client.ClientInfo{
	InterfaceName: "benchmark.BenchmarkService",
	MethodNames:   []string{"UnaryCall", "StreamCall"},
	ConnectionInjectFunc: func(dubboCliRaw any, conn *client.Connection) {
		dubboCli := dubboCliRaw.(*TripleBenchmarkServiceImpl)
		dubboCli.conn = conn
	},
}

type TripleBenchmarkServiceHandler interface {
	UnaryCall(context.Context, *BenchmarkRequest) (*BenchmarkResponse, error)
	StreamCall(context.Context, TripleBenchmarkService_StreamCallServer) error
}

func RegisterTripleBenchmarkServiceHandler(srv *server.Server, hdlr TripleBenchmarkServiceHandler, opts ...server.ServiceOption) error {
	return srv.Register(hdlr, &TripleBenchmarkService_ServiceInfo, opts...)
}

func SetTripleProviderService(srv common.RPCService) {
	dubbo.SetProviderServiceWithInfo(srv, &TripleBenchmarkService_ServiceInfo)
}

type TripleBenchmarkService_StreamCallServer interface {
	Send(*BenchmarkResponse) error
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	Conn() triple_protocol.StreamingHandlerConn
}

type TripleBenchmarkServiceStreamCallServer struct {
	*triple_protocol.BidiStream
}

func (g *TripleBenchmarkServiceStreamCallServer) Send(msg *BenchmarkResponse) error {
	return g.BidiStream.Send(msg)
}

var TripleBenchmarkService_ServiceInfo = server.ServiceInfo{
	InterfaceName: "benchmark.BenchmarkService",
	ServiceType:   (*TripleBenchmarkServiceHandler)(nil),
	Methods: []server.MethodInfo{
		{
			Name: "UnaryCall",
			Type: constant.CallUnary,
			ReqInitFunc: func() any {
				return new(BenchmarkRequest)
			},
			MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
				req := args[0].(*BenchmarkRequest)
				res, err := handler.(TripleBenchmarkServiceHandler).UnaryCall(ctx, req)
				if err != nil {
					return nil, err
				}
				return triple_protocol.NewResponse(res), nil
			},
		},
		{
			Name: "StreamCall",
			Type: constant.CallBidiStream,
			StreamInitFunc: func(baseStream any) any {
				return &TripleBenchmarkServiceStreamCallServer{baseStream.(*triple_protocol.BidiStream)}
			},
			MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
				stream := args[0].(TripleBenchmarkService_StreamCallServer)
				if err := handler.(TripleBenchmarkServiceHandler).StreamCall(ctx, stream); err != nil {
					return nil, err
				}
				return nil, nil
			},
		},
	},
}
