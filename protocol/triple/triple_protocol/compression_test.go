// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package triple_protocol

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

import (
	"google.golang.org/protobuf/types/known/emptypb"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

func TestAcceptEncodingOrdering(t *testing.T) {
	t.Parallel()
	const (
		compressionBrotli = "br"
		expect            = compressionGzip + "," + compressionBrotli
	)

	withFakeBrotli, ok := withGzip().(*compressionOption)
	assert.True(t, ok)
	withFakeBrotli.Name = compressionBrotli

	var called bool
	verify := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := r.Header.Get(tripleUnaryHeaderAcceptCompression)
		assert.Equal(t, got, expect)
		w.WriteHeader(http.StatusOK)
		called = true
	})
	server := httptest.NewServer(verify)
	defer server.Close()

	client := NewClient(
		server.Client(),
		server.URL,
		WithTriple(),
		withFakeBrotli,
		withGzip(),
	)
	_ = client.CallUnary(context.Background(), NewRequest(&emptypb.Empty{}), NewResponse(&emptypb.Empty{}))
	assert.True(t, called)
}

func TestClientCompressionOptionTest(t *testing.T) {
	t.Parallel()
	const testURL = "http://foo.bar.com/service/method"

	checkPools := func(t *testing.T, config *clientConfig) {
		t.Helper()
		assert.Equal(t, len(config.CompressionNames), len(config.CompressionPools))
		for _, name := range config.CompressionNames {
			pool := config.CompressionPools[name]
			assert.NotNil(t, pool)
		}
	}
	dummyDecompressCtor := func() Decompressor { return nil }
	dummyCompressCtor := func() Compressor { return nil }

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()
		config, err := newClientConfig(testURL, nil)
		assert.Nil(t, err)
		assert.Equal(t, config.CompressionNames, []string{compressionGzip})
		checkPools(t, config)
	})
	t.Run("WithAcceptCompression", func(t *testing.T) {
		t.Parallel()
		opts := []ClientOption{WithAcceptCompression("foo", dummyDecompressCtor, dummyCompressCtor)}
		config, err := newClientConfig(testURL, opts)
		assert.Nil(t, err)
		assert.Equal(t, config.CompressionNames, []string{compressionGzip, "foo"})
		checkPools(t, config)
	})
	t.Run("WithAcceptCompression-empty-name-noop", func(t *testing.T) {
		t.Parallel()
		opts := []ClientOption{WithAcceptCompression("", dummyDecompressCtor, dummyCompressCtor)}
		config, err := newClientConfig(testURL, opts)
		assert.Nil(t, err)
		assert.Equal(t, config.CompressionNames, []string{compressionGzip})
		checkPools(t, config)
	})
	t.Run("WithAcceptCompression-nil-ctors-noop", func(t *testing.T) {
		t.Parallel()
		opts := []ClientOption{WithAcceptCompression("foo", nil, nil)}
		config, err := newClientConfig(testURL, opts)
		assert.Nil(t, err)
		assert.Equal(t, config.CompressionNames, []string{compressionGzip})
		checkPools(t, config)
	})
	t.Run("WithAcceptCompression-nil-ctors-unregisters", func(t *testing.T) {
		t.Parallel()
		opts := []ClientOption{WithAcceptCompression("gzip", nil, nil)}
		config, err := newClientConfig(testURL, opts)
		assert.Nil(t, err)
		assert.Equal(t, config.CompressionNames, []string(nil))
		checkPools(t, config)
	})
}

func TestHandlerCompressionOptionTest(t *testing.T) {
	t.Parallel()
	const testProc = "/service/method"

	checkPools := func(t *testing.T, config *handlerConfig) {
		t.Helper()
		assert.Equal(t, len(config.CompressionNames), len(config.CompressionPools))
		for _, name := range config.CompressionNames {
			pool := config.CompressionPools[name]
			assert.NotNil(t, pool)
		}
	}
	dummyDecompressCtor := func() Decompressor { return nil }
	dummyCompressCtor := func() Compressor { return nil }

	t.Run("defaults", func(t *testing.T) {
		t.Parallel()
		config := newHandlerConfig(testProc, nil)
		assert.Equal(t, config.CompressionNames, []string{compressionGzip})
		checkPools(t, config)
	})
	t.Run("WithCompression", func(t *testing.T) {
		t.Parallel()
		opts := []HandlerOption{WithCompression("foo", dummyDecompressCtor, dummyCompressCtor)}
		config := newHandlerConfig(testProc, opts)
		assert.Equal(t, config.CompressionNames, []string{compressionGzip, "foo"})
		checkPools(t, config)
	})
	t.Run("WithCompression-empty-name-noop", func(t *testing.T) {
		t.Parallel()
		opts := []HandlerOption{WithCompression("", dummyDecompressCtor, dummyCompressCtor)}
		config := newHandlerConfig(testProc, opts)
		assert.Equal(t, config.CompressionNames, []string{compressionGzip})
		checkPools(t, config)
	})
	t.Run("WithCompression-nil-ctors-noop", func(t *testing.T) {
		t.Parallel()
		opts := []HandlerOption{WithCompression("foo", nil, nil)}
		config := newHandlerConfig(testProc, opts)
		assert.Equal(t, config.CompressionNames, []string{compressionGzip})
		checkPools(t, config)
	})
	t.Run("WithCompression-nil-ctors-unregisters", func(t *testing.T) {
		t.Parallel()
		opts := []HandlerOption{WithCompression("gzip", nil, nil)}
		config := newHandlerConfig(testProc, opts)
		assert.Equal(t, config.CompressionNames, []string(nil))
		checkPools(t, config)
	})
}

// mockDecompressor implements Decompressor for testing
type mockDecompressor struct {
	reader   io.Reader
	closed   bool
	resetErr error
	closeErr error
	readErr  error
}

func (m *mockDecompressor) Read(p []byte) (n int, err error) {
	if m.readErr != nil {
		return 0, m.readErr
	}
	return m.reader.Read(p)
}

func (m *mockDecompressor) Close() error {
	m.closed = true
	return m.closeErr
}

func (m *mockDecompressor) Reset(r io.Reader) error {
	m.reader = r
	m.closed = false
	return m.resetErr
}

// mockCompressor implements Compressor for testing
type mockCompressor struct {
	writer   io.Writer
	closed   bool
	closeErr error
	writeErr error
}

func (m *mockCompressor) Write(p []byte) (n int, err error) {
	if m.writeErr != nil {
		return 0, m.writeErr
	}
	return m.writer.Write(p)
}

func (m *mockCompressor) Close() error {
	m.closed = true
	return m.closeErr
}

func (m *mockCompressor) Reset(w io.Writer) {
	m.writer = w
	m.closed = false
}

// mockDecompressorWithDiscardError returns error when trying to read more after limit
type mockDecompressorWithDiscardError struct {
	readCount int
}

func (m *mockDecompressorWithDiscardError) Read(p []byte) (n int, err error) {
	m.readCount++
	if m.readCount == 1 {
		// First read returns data that exceeds limit
		data := []byte("hello world")
		copy(p, data)
		return len(data), nil
	}
	// Subsequent reads return error
	return 0, errors.New("discard error")
}

func (m *mockDecompressorWithDiscardError) Close() error {
	return nil
}

func (m *mockDecompressorWithDiscardError) Reset(r io.Reader) error {
	m.readCount = 0
	return nil
}

// TestNewCompressionPool tests newCompressionPool function
func TestNewCompressionPool(t *testing.T) {
	t.Parallel()

	t.Run("BothNil", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(nil, nil)
		assert.Nil(t, pool)
	})

	t.Run("WithDecompressorOnly", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor { return &mockDecompressor{} },
			nil,
		)
		assert.NotNil(t, pool)
	})

	t.Run("WithCompressorOnly", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			nil,
			func() Compressor { return &mockCompressor{} },
		)
		assert.NotNil(t, pool)
	})

	t.Run("WithBoth", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor { return &mockDecompressor{} },
			func() Compressor { return &mockCompressor{} },
		)
		assert.NotNil(t, pool)
	})
}

// TestCompressionPoolDecompress tests compressionPool.Decompress
func TestCompressionPoolDecompress(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		testData := "hello world"
		pool := newCompressionPool(
			func() Decompressor {
				return &mockDecompressor{reader: strings.NewReader(testData)}
			},
			func() Compressor { return &mockCompressor{} },
		)

		src := bytes.NewBuffer([]byte(testData))
		dst := &bytes.Buffer{}
		err := pool.Decompress(dst, src, 0)
		assert.Nil(t, err)
		assert.Equal(t, dst.String(), testData)
	})

	t.Run("WithReadMaxBytes", func(t *testing.T) {
		t.Parallel()
		testData := "hello"
		pool := newCompressionPool(
			func() Decompressor {
				return &mockDecompressor{reader: strings.NewReader(testData)}
			},
			func() Compressor { return &mockCompressor{} },
		)

		src := bytes.NewBuffer([]byte(testData))
		dst := &bytes.Buffer{}
		err := pool.Decompress(dst, src, 100)
		assert.Nil(t, err)
	})

	t.Run("ExceedsMaxBytes", func(t *testing.T) {
		t.Parallel()
		testData := "hello world this is a long message"
		pool := newCompressionPool(
			func() Decompressor {
				return &mockDecompressor{reader: strings.NewReader(testData)}
			},
			func() Compressor { return &mockCompressor{} },
		)

		src := bytes.NewBuffer([]byte(testData))
		dst := &bytes.Buffer{}
		err := pool.Decompress(dst, src, 5)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeResourceExhausted)
	})

	t.Run("ExceedsMaxBytes_DiscardError", func(t *testing.T) {
		t.Parallel()
		// Create a decompressor that returns error when reading more data
		pool := newCompressionPool(
			func() Decompressor {
				return &mockDecompressorWithDiscardError{}
			},
			func() Compressor { return &mockCompressor{} },
		)

		src := bytes.NewBuffer([]byte("test"))
		dst := &bytes.Buffer{}
		err := pool.Decompress(dst, src, 3)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeResourceExhausted)
	})

	t.Run("ResetError", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor {
				return &mockDecompressor{resetErr: errors.New("reset error")}
			},
			func() Compressor { return &mockCompressor{} },
		)

		src := bytes.NewBuffer([]byte("test"))
		dst := &bytes.Buffer{}
		err := pool.Decompress(dst, src, 0)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeInvalidArgument)
	})

	t.Run("ReadError", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor {
				return &mockDecompressor{
					reader:  strings.NewReader(""),
					readErr: errors.New("read error"),
				}
			},
			func() Compressor { return &mockCompressor{} },
		)

		src := bytes.NewBuffer([]byte("test"))
		dst := &bytes.Buffer{}
		err := pool.Decompress(dst, src, 0)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeInvalidArgument)
	})

	t.Run("CloseError", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor {
				return &mockDecompressor{
					reader:   strings.NewReader("test"),
					closeErr: errors.New("close error"),
				}
			},
			func() Compressor { return &mockCompressor{} },
		)

		src := bytes.NewBuffer([]byte("test"))
		dst := &bytes.Buffer{}
		err := pool.Decompress(dst, src, 0)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeUnknown)
	})
}

// TestCompressionPoolCompress tests compressionPool.Compress
func TestCompressionPoolCompress(t *testing.T) {
	t.Parallel()

	t.Run("Success", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor { return &mockDecompressor{} },
			func() Compressor { return &mockCompressor{} },
		)

		src := bytes.NewBuffer([]byte("hello world"))
		dst := &bytes.Buffer{}
		err := pool.Compress(dst, src)
		assert.Nil(t, err)
	})

	t.Run("WriteError", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor { return &mockDecompressor{} },
			func() Compressor {
				return &mockCompressor{writeErr: errors.New("write error")}
			},
		)

		src := bytes.NewBuffer([]byte("hello world"))
		dst := &bytes.Buffer{}
		err := pool.Compress(dst, src)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeInternal)
	})

	t.Run("CloseError", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor { return &mockDecompressor{} },
			func() Compressor {
				return &mockCompressor{closeErr: errors.New("close error")}
			},
		)

		src := bytes.NewBuffer([]byte("hello world"))
		dst := &bytes.Buffer{}
		err := pool.Compress(dst, src)
		assert.NotNil(t, err)
		assert.Equal(t, err.Code(), CodeInternal)
	})
}

// TestNamedCompressionPools tests namedCompressionPools methods
func TestNamedCompressionPools(t *testing.T) {
	t.Parallel()

	t.Run("Get_ExistingPool", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor { return &mockDecompressor{} },
			func() Compressor { return &mockCompressor{} },
		)
		pools := newReadOnlyCompressionPools(
			map[string]*compressionPool{"gzip": pool},
			[]string{"gzip"},
		)
		got := pools.Get("gzip")
		assert.NotNil(t, got)
	})

	t.Run("Get_NonExistingPool", func(t *testing.T) {
		t.Parallel()
		pools := newReadOnlyCompressionPools(
			map[string]*compressionPool{},
			[]string{},
		)
		got := pools.Get("nonexistent")
		assert.Nil(t, got)
	})

	t.Run("Get_EmptyName", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor { return &mockDecompressor{} },
			func() Compressor { return &mockCompressor{} },
		)
		pools := newReadOnlyCompressionPools(
			map[string]*compressionPool{"": pool},
			[]string{""},
		)
		got := pools.Get("")
		assert.Nil(t, got)
	})

	t.Run("Get_Identity", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor { return &mockDecompressor{} },
			func() Compressor { return &mockCompressor{} },
		)
		pools := newReadOnlyCompressionPools(
			map[string]*compressionPool{compressionIdentity: pool},
			[]string{compressionIdentity},
		)
		got := pools.Get(compressionIdentity)
		assert.Nil(t, got)
	})

	t.Run("Contains_True", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor { return &mockDecompressor{} },
			func() Compressor { return &mockCompressor{} },
		)
		pools := newReadOnlyCompressionPools(
			map[string]*compressionPool{"gzip": pool},
			[]string{"gzip"},
		)
		assert.True(t, pools.Contains("gzip"))
	})

	t.Run("Contains_False", func(t *testing.T) {
		t.Parallel()
		pools := newReadOnlyCompressionPools(
			map[string]*compressionPool{},
			[]string{},
		)
		assert.False(t, pools.Contains("nonexistent"))
	})

	t.Run("CommaSeparatedNames", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor { return &mockDecompressor{} },
			func() Compressor { return &mockCompressor{} },
		)
		pools := newReadOnlyCompressionPools(
			map[string]*compressionPool{"gzip": pool, "br": pool},
			[]string{"br", "gzip"},
		)
		names := pools.CommaSeparatedNames()
		assert.True(t, strings.Contains(names, "gzip"))
		assert.True(t, strings.Contains(names, "br"))
	})

	t.Run("CommaSeparatedNames_Empty", func(t *testing.T) {
		t.Parallel()
		pools := newReadOnlyCompressionPools(
			map[string]*compressionPool{},
			[]string{},
		)
		assert.Equal(t, pools.CommaSeparatedNames(), "")
	})
}

// TestNewReadOnlyCompressionPools tests deduplication logic
func TestNewReadOnlyCompressionPools(t *testing.T) {
	t.Parallel()

	t.Run("Deduplication", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor { return &mockDecompressor{} },
			func() Compressor { return &mockCompressor{} },
		)
		// Reversed names with duplicates
		pools := newReadOnlyCompressionPools(
			map[string]*compressionPool{"gzip": pool, "br": pool},
			[]string{"gzip", "br", "gzip", "br"},
		)
		names := pools.CommaSeparatedNames()
		// Should be deduplicated, last registered first
		assert.Equal(t, names, "br,gzip")
	})

	t.Run("ReverseOrder", func(t *testing.T) {
		t.Parallel()
		pool := newCompressionPool(
			func() Decompressor { return &mockDecompressor{} },
			func() Compressor { return &mockCompressor{} },
		)
		pools := newReadOnlyCompressionPools(
			map[string]*compressionPool{"a": pool, "b": pool, "c": pool},
			[]string{"a", "b", "c"},
		)
		// Last registered (c) should be first
		assert.Equal(t, pools.CommaSeparatedNames(), "c,b,a")
	})
}

// TestCompressionConstants tests compression constants
func TestCompressionConstants(t *testing.T) {
	t.Parallel()

	assert.Equal(t, compressionGzip, "gzip")
	assert.Equal(t, compressionIdentity, "identity")
}

// TestCompressionPoolGetDecompressorTypeError tests type assertion error path
func TestCompressionPoolGetDecompressorTypeError(t *testing.T) {
	t.Parallel()

	// Create a pool that returns wrong type
	pool := &compressionPool{}
	pool.decompressors.New = func() any { return "not a decompressor" }
	pool.compressors.New = func() any { return &mockCompressor{} }

	src := bytes.NewBuffer([]byte("test"))
	dst := &bytes.Buffer{}
	err := pool.Decompress(dst, src, 0)
	assert.NotNil(t, err)
	assert.Equal(t, err.Code(), CodeInvalidArgument)
	assert.True(t, strings.Contains(err.Message(), "expected Decompressor"))
}

// TestCompressionPoolGetCompressorTypeError tests type assertion error path
func TestCompressionPoolGetCompressorTypeError(t *testing.T) {
	t.Parallel()

	// Create a pool that returns wrong type
	pool := &compressionPool{}
	pool.decompressors.New = func() any { return &mockDecompressor{} }
	pool.compressors.New = func() any { return "not a compressor" }

	src := bytes.NewBuffer([]byte("test"))
	dst := &bytes.Buffer{}
	err := pool.Compress(dst, src)
	assert.NotNil(t, err)
	assert.Equal(t, err.Code(), CodeUnknown)
	assert.True(t, strings.Contains(err.Message(), "expected Compressor"))
}

// TestGzipRoundTrip tests actual gzip compression/decompression
func TestGzipRoundTrip(t *testing.T) {
	t.Parallel()

	// Create a real gzip compression pool
	pool := newCompressionPool(
		func() Decompressor {
			return &gzipDecompressor{reader: new(gzip.Reader)}
		},
		func() Compressor {
			w, _ := gzip.NewWriterLevel(io.Discard, gzip.BestSpeed)
			return w
		},
	)

	testData := []byte("hello world, this is a test message for compression")

	// Compress
	src := bytes.NewBuffer(testData)
	compressed := &bytes.Buffer{}
	err := pool.Compress(compressed, src)
	assert.Nil(t, err)

	// Decompress
	decompressed := &bytes.Buffer{}
	err = pool.Decompress(decompressed, compressed, 0)
	assert.Nil(t, err)
	assert.Equal(t, decompressed.Bytes(), testData)
}

// gzipDecompressor wraps gzip.Reader to implement Decompressor
type gzipDecompressor struct {
	reader *gzip.Reader
}

func (g *gzipDecompressor) Read(p []byte) (n int, err error) {
	return g.reader.Read(p)
}

func (g *gzipDecompressor) Close() error {
	return g.reader.Close()
}

func (g *gzipDecompressor) Reset(r io.Reader) error {
	return g.reader.Reset(r)
}
