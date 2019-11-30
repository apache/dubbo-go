/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package grpc

import (
	"net"
)

import (
	"google.golang.org/grpc"
)

import (
	"github.com/apache/dubbo-go/common"
)

type Server struct {
	grpcServer *grpc.Server
}

func NewServer() *Server {

	return nil
}

// TODO: unimplemented
func (s *Server) Start(url common.URL) {
	var (
		addr string
		err  error
	)
	addr = url.Location
	lis, err := net.Listen("tcp", addr)
	if err != nil {
		panic(err)
	}
	server := grpc.NewServer()

	s.grpcServer = server
	// grpc-go 必须提前注册
	// ServiceDesc 这个信息需要有
	// 需要找一个方法。
	//server.RegisterService()
	// 想个办法注册下
	if err = server.Serve(lis); err != nil {
		panic(err)
	}
}

func (s *Server) Stop() {
	s.grpcServer.Stop()
}
