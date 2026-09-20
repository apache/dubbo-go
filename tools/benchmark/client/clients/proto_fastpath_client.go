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

package clients

import (
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

import (
	"google.golang.org/protobuf/proto"
)

import (
	benchmark "dubbo.apache.org/dubbo-go/v3/tools/benchmark/proto"
)

// ProtoFastPathClient is the prototype-shaped unary client: it talks raw
// HTTP/2 (h2c) to the Triple server and performs a unary call without going
// through duplexHTTPCall -- no io.Pipe, no per-request goroutine. It mirrors
// fastPathUnaryPing in unary_fastpath_bench_test.go, so pressure-testing it
// measures the same "fast path design" the prototype benchmark measures.
type ProtoFastPathClient struct {
	httpClient *http.Client
	url        string
	payload    []byte
	pool       *sync.Pool
}

// NewProtoFastPathClient builds a ProtoFastPathClient against a Triple server
// at addr. It talks plain HTTP/1.1 with no automatic gzip (DisableCompression),
// matching the wire shape of DubboGoClient (compression=none) so the A/B is
// fair; the fast-path shape itself -- no io.Pipe, no per-request goroutine,
// pooled response buffer, Content-Length -- is transport-independent.
func NewProtoFastPathClient(addr string, payload []byte) *ProtoFastPathClient {
	transport := &http.Transport{DisableCompression: true}
	return &ProtoFastPathClient{
		httpClient: &http.Client{Transport: transport},
		url:        "http://" + addr + "/benchmark.BenchmarkService/UnaryCall",
		payload:    payload,
		pool: &sync.Pool{
			New: func() any {
				return bytes.NewBuffer(make([]byte, 0, 512))
			},
		},
	}
}

// fastPathBody is a zero-extra-allocation io.ReadCloser mirroring the
// production unaryRequestBody: it feeds the request body from the marshaled
// payload without wrapping bytes.Reader in io.NopCloser (net/http would
// otherwise allocate both the *bytes.Reader and the NopCloser wrapper per
// call). This is the fourth optimization (beyond O1/O2/O3) that flips the
// remaining allocs/B/op side effects positive.
type fastPathBody struct {
	data []byte
}

func (b *fastPathBody) Read(p []byte) (int, error) {
	if len(b.data) == 0 {
		return 0, io.EOF
	}
	n := copy(p, b.data)
	b.data = b.data[n:]
	return n, nil
}

func (b *fastPathBody) Close() error {
	return nil
}

// Call sends one unary request through the fast-path shape.
func (c *ProtoFastPathClient) Call(ctx context.Context) error {
	reqBuf := c.pool.Get().(*bytes.Buffer)
	data, err := proto.MarshalOptions{}.MarshalAppend(reqBuf.Bytes()[:0], &benchmark.BenchmarkRequest{Payload: c.payload})
	if err != nil {
		c.put(reqBuf)
		return fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.url, &fastPathBody{data: data})
	if err != nil {
		c.put(reqBuf)
		return fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Content-Type", "application/proto")
	// No Accept-Encoding: gzip: the A/B baseline DubboGoClient is measured with
	// compression=none (the benchmark tool's default), so the fast-path client
	// must negotiate the same no-compression wire shape for a fair comparison.
	req.ContentLength = int64(len(data))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.put(reqBuf)
		return fmt.Errorf("do request: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		c.put(reqBuf)
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	respBuf := c.pool.Get().(*bytes.Buffer)
	body := io.Reader(resp.Body)
	if strings.EqualFold(resp.Header.Get("Content-Encoding"), "gzip") {
		gz, gzErr := gzip.NewReader(body)
		if gzErr != nil {
			resp.Body.Close()
			c.put(reqBuf)
			c.put(respBuf)
			return fmt.Errorf("new gzip reader: %w", gzErr)
		}
		defer gz.Close()
		body = gz
	}
	if _, err := respBuf.ReadFrom(body); err != nil {
		resp.Body.Close()
		c.put(reqBuf)
		c.put(respBuf)
		return fmt.Errorf("read response: %w", err)
	}
	resp.Body.Close()
	// Unary semantics: the server only responds after consuming the full
	// request body, so reqBuf can be recycled here.
	c.put(reqBuf)

	var out benchmark.BenchmarkResponse
	if err := proto.Unmarshal(respBuf.Bytes(), &out); err != nil {
		c.put(respBuf)
		return fmt.Errorf("unmarshal response: %w", err)
	}
	c.put(respBuf)
	return nil
}

// Close releases resources held by the client.
func (c *ProtoFastPathClient) Close() error {
	c.httpClient.CloseIdleConnections()
	return nil
}

// String returns a short description for logging.
func (c *ProtoFastPathClient) String() string {
	return "proto-fastpath (raw h2c, no duplexHTTPCall)"
}

func (c *ProtoFastPathClient) put(buf *bytes.Buffer) {
	if buf.Cap() > 8*1024*1024 {
		return
	}
	buf.Reset()
	c.pool.Put(buf)
}
