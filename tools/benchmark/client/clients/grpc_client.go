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

package clients

import (
	"context"
	"fmt"
)

import (
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

import (
	benchmark "dubbo.apache.org/dubbo-go/v3/tools/benchmark/proto"
)

type GrpcClient struct {
	conn     *grpc.ClientConn
	client   benchmark.BenchmarkServiceClient
	payload  []byte
	callMode string
}

func NewGrpcClient(addr string, callMode string, payload []byte) (*GrpcClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("连接gRPC服务失败: %v", err)
	}

	client := benchmark.NewBenchmarkServiceClient(conn)

	return &GrpcClient{
		conn:     conn,
		client:   client,
		payload:  payload,
		callMode: callMode,
	}, nil
}

func (c *GrpcClient) Call(ctx context.Context) error {
	switch c.callMode {
	case "unary":
		return c.unaryCall(ctx)
	case "streaming":
		return c.streamCall(ctx)
	default:
		return c.unaryCall(ctx)
	}
}

func (c *GrpcClient) unaryCall(ctx context.Context) error {
	req := &benchmark.BenchmarkRequest{Payload: c.payload}
	_, err := c.client.UnaryCall(ctx, req)
	return err
}

func (c *GrpcClient) streamCall(ctx context.Context) error {
	stream, err := c.client.StreamCall(ctx)
	if err != nil {
		return err
	}

	req := &benchmark.BenchmarkRequest{Payload: c.payload}
	if err := stream.Send(req); err != nil {
		return err
	}

	_, err = stream.Recv()
	return err
}

func (c *GrpcClient) Close() error {
	return c.conn.Close()
}

func (c *GrpcClient) String() string {
	return fmt.Sprintf("gRPC Client: callMode=%s, payloadSize=%d", c.callMode, len(c.payload))
}
