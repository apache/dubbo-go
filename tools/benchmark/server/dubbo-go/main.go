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
	"io"
	"os"
	"os/signal"
	"syscall"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
	"dubbo.apache.org/dubbo-go/v3/server"
	benchmark "dubbo.apache.org/dubbo-go/v3/tools/benchmark/proto"
)

const separator = "========================================"

var (
	serialization = flag.String("serialization", "protobuf", "Serialization protocol: hessian2 / protobuf / msgpack")
	compression   = flag.String("compression", "none", "Compression strategy: none / default / fastest")
	port          = flag.Int("port", 20000, "Server port")
)

type BenchmarkServiceImpl struct {
	benchmark.TripleBenchmarkServiceHandler
}

func (s *BenchmarkServiceImpl) UnaryCall(ctx context.Context, req *benchmark.BenchmarkRequest) (*benchmark.BenchmarkResponse, error) {
	return &benchmark.BenchmarkResponse{Payload: req.Payload}, nil
}

func (s *BenchmarkServiceImpl) StreamCall(ctx context.Context, stream benchmark.TripleBenchmarkService_StreamCallServer) error {
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
	logger.Info("    Dubbo-Go Benchmark Server")
	logger.Info(separator)
	logger.Infof("[INFO] Serialization: %s", *serialization)
	logger.Infof("[INFO] Compression:   %s", *compression)
	logger.Infof("[INFO] Port:          %d", *port)

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
		logger.Fatalf("Failed to create server: %v", err)
	}

	if err := benchmark.RegisterTripleBenchmarkServiceHandler(srv, &BenchmarkServiceImpl{}); err != nil {
		logger.Fatalf("Failed to register service: %v", err)
	}

	go func() {
		if err := srv.Serve(); err != nil {
			logger.Fatalf("Failed to start server: %v", err)
		}
	}()

	logger.Infof("[INFO] Server started, listening on: 127.0.0.1:%d", *port)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	logger.Info("[INFO] Stopping server...")
	if err := graceful_shutdown.Shutdown(context.Background()); err != nil {
		logger.Errorf("Failed to stop server: %v", err)
	}
	logger.Info("[INFO] Server stopped")
	os.Exit(0)
}
