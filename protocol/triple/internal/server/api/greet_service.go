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

package api

import (
	"context"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/greettriple"
	"errors"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	"io"
	"strings"
)

type GreetConnectServer struct {
}

func (srv *GreetConnectServer) Greet(ctx context.Context, req *greet.GreetRequest) (*greet.GreetResponse, error) {
	resp := &greet.GreetResponse{Greeting: req.Name}
	return resp, nil
}

func (srv *GreetConnectServer) GreetStream(ctx context.Context, stream greettriple.GreetService_GreetStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("connect recv error: %s", err)
		}
		if err := stream.Send(&greet.GreetStreamResponse{Greeting: req.Name}); err != nil {
			return fmt.Errorf("connect send error: %s", err)
		}
	}
}

func (srv *GreetConnectServer) GreetClientStream(ctx context.Context, stream greettriple.GreetService_GreetClientStreamServer) (*greet.GreetClientStreamResponse, error) {
	var reqs []string
	for stream.Recv() {
		reqs = append(reqs, stream.Msg().Name)
	}
	if stream.Err() != nil && !errors.Is(stream.Err(), io.EOF) {
		logger.Errorf("ClientStream unexpected err: %s", stream.Err())
	}
	resp := &greet.GreetClientStreamResponse{
		Greeting: strings.Join(reqs, ","),
	}

	return resp, nil
}

func (srv *GreetConnectServer) GreetServerStream(ctx context.Context, req *greet.GreetServerStreamRequest, stream greettriple.GreetService_GreetServerStreamServer) error {
	for i := 0; i < 5; i++ {
		if err := stream.Send(&greet.GreetServerStreamResponse{Greeting: "Hello" + req.Name}); err != nil {
			logger.Errorf("ServerStream unexpected err: %s", err)
		}
	}
	return nil
}
