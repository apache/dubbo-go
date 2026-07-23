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
)

import (
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	_ "dubbo.apache.org/dubbo-go/v3/imports" // import required for dubbo-go initialization
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
	"dubbo.apache.org/dubbo-go/v3/server"
	benchmark "dubbo.apache.org/dubbo-go/v3/tools/benchmark/proto/benchmark_gen"
)

const separator = "========================================"

var (
	serialization = flag.String("serialization", "protobuf", "序列化协议: hessian2 / protobuf / msgpack")
	compression   = flag.String("compression", "none", "压缩策略: none / default / fastest")
	port          = flag.Int("port", 20000, "服务端口")
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
	fmt.Printf("[INFO] 序列化:   %s\n", *serialization)
	fmt.Printf("[INFO] 压缩:     %s\n", *compression)
	fmt.Printf("[INFO] 端口:     %d\n", *port)

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
		log.Fatalf("创建服务失败: %v", err)
	}

	if err := benchmark.RegisterTripleBenchmarkServiceHandler(srv, &BenchmarkServiceImpl{}); err != nil {
		log.Fatalf("注册服务失败: %v", err)
	}

	go func() {
		if err := srv.Serve(); err != nil {
			log.Fatalf("服务启动失败: %v", err)
		}
	}()

	fmt.Printf("[INFO] 服务已启动，监听: 127.0.0.1:%d\n", *port)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig

	fmt.Println("[INFO] 正在停止服务...")
	if err := graceful_shutdown.Shutdown(context.Background()); err != nil {
		log.Printf("[WARN] 服务停止失败: %v", err)
	}
	fmt.Println("[INFO] 服务已停止")
}
