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

package triple_protocol

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/rpcerror"
)

func TestCompressionAttribution(t *testing.T) {
	pool := newCompressionPool(func() Decompressor { return &gzip.Reader{} }, func() Compressor { return gzip.NewWriter(nil) })
	var dst bytes.Buffer
	err := pool.Decompress(&dst, bytes.NewBufferString("invalid gzip"), 0)
	if got := rpcerror.Classify(err); got.Category != rpcerror.CategoryCodec || got.Origin != rpcerror.Codec {
		t.Fatalf("corruption: %+v %v", got, err)
	}
	var encoded bytes.Buffer
	if compressErr := pool.Compress(&encoded, bytes.NewBufferString("hello world")); compressErr != nil {
		t.Fatal(compressErr)
	}
	dst.Reset()
	err = pool.Decompress(&dst, &encoded, 2)
	if got := rpcerror.Classify(err); got.Category != rpcerror.CategoryLimit || got.Origin != rpcerror.Unknown {
		t.Fatalf("limit: %+v %v", got, err)
	}
}

type corruptCompressor struct{}

func (corruptCompressor) Reset(io.Writer)           {}
func (corruptCompressor) Write([]byte) (int, error) { return 0, errors.New("corrupt encoder") }
func (corruptCompressor) Close() error              { return nil }

func TestCompressorAndEnvelopeAttribution(t *testing.T) {
	pool := newCompressionPool(func() Decompressor { return &gzip.Reader{} }, func() Compressor { return corruptCompressor{} })
	var dst bytes.Buffer
	err := pool.Compress(&dst, bytes.NewBufferString("payload"))
	if rpcerror.Classify(err).Category != rpcerror.CategoryCodec {
		t.Fatal(err)
	}
	writer := envelopeWriter{writer: &dst, sendMaxBytes: 2}
	err = writer.Write(&envelope{Data: bytes.NewBufferString("payload")})
	if rpcerror.Classify(err).Category != rpcerror.CategoryLimit || rpcerror.Of(err) != rpcerror.Unknown {
		t.Fatal(err)
	}
	// Header succeeds but checksum fails during decompression.
	var compressed bytes.Buffer
	gz := gzip.NewWriter(&compressed)
	gz.Write([]byte("payload"))
	gz.Close()
	data := compressed.Bytes()
	data[len(data)-1] ^= 1
	err = pool.Decompress(&dst, bytes.NewBuffer(data), 0)
	if rpcerror.Classify(err).Category != rpcerror.CategoryCodec {
		t.Fatal(err)
	}
}
