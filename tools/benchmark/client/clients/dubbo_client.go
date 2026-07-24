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
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	_ "dubbo.apache.org/dubbo-go/v3/imports" // import required for dubbo-go initialization
	benchmark "dubbo.apache.org/dubbo-go/v3/tools/benchmark/proto"
)

type DubboGoClient struct {
	client   benchmark.TripleBenchmarkService
	payload  []byte
	callMode string
}

func NewDubboGoClient(addr string, serialization, compression, callMode string, payload []byte) (*DubboGoClient, error) {
	cli, err := client.NewClient(
		client.WithClientNoCheck(),
		client.WithClientSerialization(serialization),
		client.WithClientParam(constant.SerializationKey, serialization),
	)
	if err != nil {
		return nil, fmt.Errorf("创建Dubbo客户端失败: %v", err)
	}

	service, err := benchmark.NewTripleBenchmarkService(cli,
		client.WithURL(fmt.Sprintf("tri://%s/%s", addr, benchmark.BenchmarkServiceName)),
		client.WithSerialization(serialization),
		client.WithParam(constant.MaxCallRecvMsgSize, "16MB"),
		client.WithParam(constant.MaxCallSendMsgSize, "16MB"),
	)
	if err != nil {
		return nil, fmt.Errorf("创建BenchmarkService失败: %v", err)
	}

	return &DubboGoClient{
		client:   service,
		payload:  payload,
		callMode: callMode,
	}, nil
}

func (c *DubboGoClient) Call(ctx context.Context) error {
	switch c.callMode {
	case "unary":
		return c.unaryCall(ctx)
	case "streaming":
		return c.streamCall(ctx)
	default:
		return c.unaryCall(ctx)
	}
}

func (c *DubboGoClient) unaryCall(ctx context.Context) error {
	req := &benchmark.BenchmarkRequest{Payload: c.payload}
	_, err := c.client.UnaryCall(ctx, req)
	return err
}

func (c *DubboGoClient) streamCall(ctx context.Context) error {
	stream, err := c.client.StreamCall(ctx)
	if err != nil {
		return err
	}
	defer stream.Close()

	req := &benchmark.BenchmarkRequest{Payload: c.payload}
	if err := stream.Send(req); err != nil {
		return err
	}

	if !stream.Recv() {
		return stream.Err()
	}

	return nil
}

func (c *DubboGoClient) Close() error {
	return nil
}

func (c *DubboGoClient) String() string {
	return fmt.Sprintf("Dubbo-Go Client: callMode=%s, payloadSize=%d", c.callMode, len(c.payload))
}
