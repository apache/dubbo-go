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

// stream_ab is an independent streaming benchmark client. It talks straight to
// a real Triple (dubbo-go) server over cleartext HTTP/2 (h2c) using only
// triple_protocol.NewClient — no dubbo-go high-level client/reference machinery
// is involved.
//
// Each worker operation opens one stream, sends msgs small messages, closes the
// request side, then drains every response. The two arms differ only in whether
// triple_protocol.WithWriteBuffering() is passed (buffered) or not (baseline).
// All wire parameters (triple protocol, default gRPC wire encoding, no request
// compression, h2c cleartext) are identical so the comparison isolates the
// variable under study.
package main

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"
)

import (
	"golang.org/x/net/http2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/tools/benchmark/client/engine"
	benchmark "dubbo.apache.org/dubbo-go/v3/tools/benchmark/proto"
)

var (
	addr        = flag.String("addr", "127.0.0.1:20000", "Server host:port")
	payloadSize = flag.Int("payload", 128, "Per-message payload size (bytes)")
	msgCount    = flag.Int("msgs", 128, "Number of messages carried by one stream")
	concurrency = flag.Int("concurrency", 50, "Concurrent worker goroutines")
	duration    = flag.String("duration", "60s", "Test duration after warmup")
	warmup      = flag.String("warmup", "10s", "Warmup duration")
	timeout     = flag.String("timeout", "90s", "Per-operation (one stream round-trip) timeout")
	buffering   = flag.Bool("buffering", false, "Enable triple_protocol.WithWriteBuffering()")
	probe       = flag.Bool("probe", false, "Run a single stream round-trip and print detailed per-step errors, then exit")
)

func main() {
	flag.Parse()

	payload := make([]byte, *payloadSize)
	if _, err := rand.Read(payload); err != nil {
		for i := range payload {
			payload[i] = byte(i % 256)
		}
	}

	warmDur := mustParseDur(*warmup, "warmup")
	testDur := mustParseDur(*duration, "duration")
	callTimeout := mustParseDur(*timeout, "timeout")

	client := newStreamClient(*addr, *buffering)

	// --probe runs exactly one stream round-trip and prints the real error, so a
	// wiring/wire-encoding problem surfaces directly instead of being swallowed
	// by the benchmark engine's metrics collector.
	if *probe {
		ctx, cancel := context.WithTimeout(context.Background(), callTimeout)
		defer cancel()
		received, err := oneStreamCall(ctx, client, payload, *msgCount)
		if err != nil {
			fmt.Fprintf(os.Stderr, "PROBE FAILED after %d/%d responses: %v (errors.Is(err, io.EOF)=%v)\n",
				received, *msgCount, err, errors.Is(err, io.EOF))
			os.Exit(1)
		}
		fmt.Printf("PROBE OK: stream round-trip with %d messages succeeded (buffering=%v)\n", *msgCount, *buffering)
		return
	}

	// Field the per-call timeout through the context so a stream that stalls
	// for longer than callTimeout fails fast instead of wedging the worker.
	benchEngine := engine.NewEngine(*concurrency, warmDur, testDur, callTimeout)

	fmt.Printf("%s\naddr=%s  buffering=%v  payload=%dB  msgs/stream=%d  concurrency=%d  duration=%s  warmup=%s\n%s\n",
		"========================================",
		*addr, *buffering, *payloadSize, *msgCount, *concurrency, testDur, warmDur,
		"========================================")

	stats := benchEngine.Run(func(ctx context.Context) (time.Duration, error) {
		start := time.Now()
		_, err := oneStreamCall(ctx, client, payload, *msgCount)
		return time.Since(start), err
	})

	fmt.Println(stats.String())
	// Each stream carries msgCount messages, so the message rate is the stream
	// rate times msgCount. Payload throughput is counted on the send side, which
	// keeps measurements with different payload sizes comparable.
	fmt.Printf("message throughput ~= %.2f msgs/s  (per-stream QPS %.2f x msgs %d)\n",
		stats.QPS*float64(*msgCount), stats.QPS, *msgCount)
	fmt.Printf("payload throughput ~= %.2f MB/s  (send side, %d B/message)\n",
		stats.QPS*float64(*msgCount)*float64(*payloadSize)/(1024*1024), *payloadSize)
}

// newStreamClient builds a Triple client over plaintext HTTP/2, mirroring the
// official dubbo-go triple client wiring (protocol/triple/client.go):
// http2.Transport with AllowHTTP + a plaintext DialTLSContext. No protocol
// option is passed, so the client uses dubbo-go's defaults: triple protocol and
// gRPC wire encoding. The default codec is
// protobuf-binary and requests are sent uncompressed, so the wire shape
// matches the benchmark server's `--serialization protobuf --compression none`.
func newStreamClient(addr string, buffering bool) *triple_protocol.Client {
	transport := &http2.Transport{
		AllowHTTP:       true,
		ReadIdleTimeout: 15 * time.Second,
		PingTimeout:     5 * time.Second,
		DialTLSContext: func(ctx context.Context, network, _ string, _ *tls.Config) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, network, addr)
		},
	}
	httpClient := &http.Client{
		Transport: transport,
		// No automatic redirect following / gzip handling; the server runs
		// compression=none and Triple handles decompression itself.
	}

	opts := []triple_protocol.ClientOption{}
	if buffering {
		opts = append(opts, triple_protocol.WithWriteBuffering())
	}

	return triple_protocol.NewClient(
		httpClient,
		"http://"+addr+"/benchmark.BenchmarkService", // nosonar
		opts...,
	)
}

// oneStreamCall sends msgs messages on a single bidirectional stream, closes the
// request side, then drains every response. It returns how many responses were
// received, so a short read is distinguishable from a protocol error.
func oneStreamCall(ctx context.Context, client *triple_protocol.Client, payload []byte, msgs int) (int, error) {
	stream, err := client.CallBidiStream(ctx, "StreamCall")
	if err != nil {
		return 0, fmt.Errorf("open stream: %w", err)
	}
	defer stream.CloseResponse()

	req := &benchmark.BenchmarkRequest{Payload: payload}
	for i := range msgs {
		if err := stream.Send(req); err != nil {
			return 0, fmt.Errorf("send %d: %w", i, err)
		}
	}
	if err := stream.CloseRequest(); err != nil {
		return 0, fmt.Errorf("close request: %w", err)
	}

	received := 0
	resp := new(benchmark.BenchmarkResponse)
	for {
		err := stream.Receive(resp)
		if errors.Is(err, io.EOF) {
			return received, nil
		}
		if err != nil {
			return received, fmt.Errorf("receive: %w", err)
		}
		received++
	}
}

// mustParseDur parses a duration flag or exits.
func mustParseDur(s, name string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid %s duration %q: %v\n", name, s, err)
		os.Exit(2)
	}
	return d
}
