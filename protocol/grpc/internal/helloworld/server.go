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

package helloworld

import (
	"context"
	"log"
	"net"
)

import (
	"google.golang.org/grpc"
)

// server is used to implement helloworld.GreeterServer.
type server struct {
	*GreeterProviderBase
}

func NewService() *server {
	return &server{
		GreeterProviderBase: &GreeterProviderBase{},
	}
}

// SayHello implements helloworld.GreeterServer
func (s *server) SayHello(ctx context.Context, in *HelloRequest) (*HelloReply, error) {
	log.Printf("Received: %v", in.GetName())
	return &HelloReply{Message: "Hello " + in.GetName()}, nil
}

func (s *server) Reference() string {
	return "GrpcGreeterImpl"
}

type Server struct {
	listener net.Listener
	server   *grpc.Server
}

func NewServer(address string) (*Server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	server := grpc.NewServer()
	service := NewService()
	RegisterGreeterServer(server, service)

	s := Server{
		listener: listener,
		server:   server,
	}
	return &s, nil
}

func (s *Server) Start() {
	if err := s.server.Serve(s.listener); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *Server) Stop() {
	s.server.GracefulStop()
}
