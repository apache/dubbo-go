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
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/dubbo3_gen"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	"github.com/pkg/errors"
	"io"
	"strings"
)

type GreetDubbo3Server struct {
	greet.UnimplementedGreetServiceServer
}

func (srv *GreetDubbo3Server) Greet(ctx context.Context, req *proto.GreetRequest) (*proto.GreetResponse, error) {
	return &proto.GreetResponse{Greeting: req.Name}, nil
}

func (srv *GreetDubbo3Server) GreetStream(stream greet.GreetService_GreetStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			logger.Errorf("Stream Recv unexpected err: %s", err)
			return fmt.Errorf("dubbo3 recv error: %s", err)
		}
		if err := stream.Send(&proto.GreetStreamResponse{Greeting: req.Name}); err != nil {
			logger.Errorf("Stream Send unexpected err: %s", err)
			return fmt.Errorf("dubbo3 send error: %s", err)
		}
	}
	return nil
}

func (srv *GreetDubbo3Server) GreetClientStream(stream greet.GreetService_GreetClientStreamServer) error {
	var reqs []string
	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			logger.Errorf("ClientStream Recv unexpected err: %s", err)
			return fmt.Errorf("dubbo3 recv error: %s", err)
		}
		reqs = append(reqs, req.Name)
	}

	resp := &proto.GreetClientStreamResponse{
		Greeting: strings.Join(reqs, ","),
	}
	return stream.SendAndClose(resp)
}

func (srv *GreetDubbo3Server) GreetServerStream(req *proto.GreetServerStreamRequest, stream greet.GreetService_GreetServerStreamServer) error {
	for i := 0; i < 5; i++ {
		if err := stream.Send(&proto.GreetServerStreamResponse{Greeting: req.Name}); err != nil {
			logger.Errorf("ServerStream Send unexpected err: %s", err)
			return err
		}
	}
	return nil
}
