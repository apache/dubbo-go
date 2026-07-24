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
	"os"
	"os/signal"
	"syscall"

	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"

	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
	"dubbo.apache.org/dubbo-go/v3/server"

	// import required for dubbo-go initialization

	benchmark "dubbo.apache.org/dubbo-go/v3/tools/benchmark/proto/benchmark_gen"
)

const separator = "========================================"

var (
	serialization = flag.String("serialization", "protobuf", "serialization protocol: hessian2 / protobuf / msgpack")
	compression   = flag.String("compression", "none", "compression strategy: none / default / fastest")
	port          = flag.Int("port", 20000, "server port")
)

type BenchmarkServiceImpl struct {
	benchmark.TripleBenchmarkServiceHandler
}

func (s *BenchmarkServiceImpl) UnaryCall(ctx context.Context, req *benchmark.BenchmarkRequest) (*benchmark.BenchmarkResponse, error) {
	return &benchmark.BenchmarkResponse{Payload: req.Payload}, nil
}

func (s *BenchmarkServiceImpl) StreamCall(ctx context.Context, stream benchmark.TripleBenchmarkService_StreamCallServer) error {
	return nil
}

func main() {
	flag.Parse()

	fmt.Println(separator)
	fmt.Println("    Dubbo-Go Benchmark Server")
	fmt.Println(separator)
	fmt.Printf("[INFO] Serialization: %s\n", *serialization)
	fmt.Printf("[INFO] Compression:   %s\n", *compression)
	fmt.Printf("[INFO] Port:          %d\n", *port)

	srv, err := server.NewServer(
		server.WithServerProtocol(
			protocol.WithTriple(
				triple.WithMaxServerRecvMsgSize("16MB"),
				triple.WithMaxServerSendMsgSize("16MB"),
			),
			protocol.WithPort(*port),
			protocol.WithParams(map[string]string{
				"serialization": *serialization,
				"compression":   *compression,
			}),
		),
	)
	if err != nil {
		log.Fatalf("failed to create server: %v", err)
	}

	if err := benchmark.RegisterTripleBenchmarkServiceHandler(srv, &BenchmarkServiceImpl{}); err != nil {
		log.Fatalf("failed to register service: %v", err)
	}

	go func() {
		if err := srv.Serve(); err != nil {
			log.Fatalf("failed to start server: %v", err)
		}
	}()

	fmt.Printf("[INFO] server started, listening on: 127.0.0.1:%d\n", *port)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("[INFO] stopping server...")
	if err := graceful_shutdown.Shutdown(context.Background()); err != nil {
		log.Printf("[WARN] failed to stop server: %v", err)
	}
	fmt.Println("[INFO] server stopped")
}
