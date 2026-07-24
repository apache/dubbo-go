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
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	benchmark "dubbo.apache.org/dubbo-go/v3/tools/benchmark/proto/benchmark_gen"
)

const separator = "========================================"

var (
	port = flag.Int("port", 50051, "server port")
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
			return nil
		}
		if err := stream.Send(&benchmark.BenchmarkResponse{Payload: req.Payload}); err != nil {
			return err
		}
	}
}

func main() {
	flag.Parse()

	fmt.Println(separator)
	fmt.Println("      gRPC Benchmark Server")
	fmt.Println(separator)
	fmt.Printf("[INFO] Port:     %d\n", *port)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	benchmark.RegisterBenchmarkServiceServer(s, &benchmarkServiceImpl{})

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	fmt.Printf("[INFO] server started, listening on: 127.0.0.1:%d\n", *port)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("[INFO] stopping server...")
	s.GracefulStop()
	fmt.Println("[INFO] server stopped")
}
