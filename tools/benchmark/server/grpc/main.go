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

package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"syscall"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"google.golang.org/grpc"
)

import (
	benchmark "dubbo.apache.org/dubbo-go/v3/tools/benchmark/proto"
)

const separator = "========================================"

var (
	port = flag.Int("port", 50051, "Server port")
)

type benchmarkServiceImpl struct {
	benchmark.UnimplementedBenchmarkServiceServer
}

func (s *benchmarkServiceImpl) UnaryCall(ctx context.Context, req *benchmark.BenchmarkRequest) (*benchmark.BenchmarkResponse, error) {
	return &benchmark.BenchmarkResponse{Payload: req.Payload}, nil
}

func (s *benchmarkServiceImpl) StreamCall(stream benchmark.BenchmarkService_StreamCallServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if err := stream.Send(&benchmark.BenchmarkResponse{Payload: req.Payload}); err != nil {
			return err
		}
	}
}

func main() {
	flag.Parse()

	logger.Info(separator)
	logger.Info("      gRPC Benchmark Server")
	logger.Info(separator)
	logger.Infof("[INFO] Port:          %d", *port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		logger.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	benchmark.RegisterBenchmarkServiceServer(s, &benchmarkServiceImpl{})

	go func() {
		if err := s.Serve(lis); err != nil {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Infof("[INFO] Server started, listening on: 127.0.0.1:%d", *port)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logger.Info("[INFO] Stopping server...")
	s.GracefulStop()
	logger.Info("[INFO] Server stopped")
}
