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
	"fmt"
	"io"
	"strings"
)

import (
	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/dubbo3_gen"
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
			return fmt.Errorf("dubbo3 Bidistream recv error: %s", err)
		}
		if err := stream.Send(&proto.GreetStreamResponse{Greeting: req.Name}); err != nil {
			return fmt.Errorf("dubbo3 Bidistream send error: %s", err)
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
			return fmt.Errorf("dubbo3 ClientStream recv error: %s", err)
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
			return fmt.Errorf("dubbo3 ServerStream send error: %s", err)
		}
	}
	return nil
}

const (
	GroupVersionIdentifier = "g1v1"
)

type GreetDubbo3ServerGroup1Version1 struct {
	greet.UnimplementedGreetServiceServer
}

func (srv *GreetDubbo3ServerGroup1Version1) Greet(ctx context.Context, req *proto.GreetRequest) (*proto.GreetResponse, error) {
	return &proto.GreetResponse{Greeting: GroupVersionIdentifier + req.Name}, nil
}

func (srv *GreetDubbo3ServerGroup1Version1) GreetStream(stream greet.GreetService_GreetStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("dubbo3 Bidistream recv error: %s", err)
		}
		if err := stream.Send(&proto.GreetStreamResponse{Greeting: GroupVersionIdentifier + req.Name}); err != nil {
			return fmt.Errorf("dubbo3 Bidistream send error: %s", err)
		}
	}
	return nil
}

func (srv *GreetDubbo3ServerGroup1Version1) GreetClientStream(stream greet.GreetService_GreetClientStreamServer) error {
	var reqs []string
	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return fmt.Errorf("dubbo3 ClientStream recv error: %s", err)
		}
		reqs = append(reqs, GroupVersionIdentifier+req.Name)
	}

	resp := &proto.GreetClientStreamResponse{
		Greeting: strings.Join(reqs, ","),
	}
	return stream.SendAndClose(resp)
}

func (srv *GreetDubbo3ServerGroup1Version1) GreetServerStream(req *proto.GreetServerStreamRequest, stream greet.GreetService_GreetServerStreamServer) error {
	for i := 0; i < 5; i++ {
		if err := stream.Send(&proto.GreetServerStreamResponse{Greeting: GroupVersionIdentifier + req.Name}); err != nil {
			return fmt.Errorf("dubbo3 ServerStream send error: %s", err)
		}
	}
	return nil
}
