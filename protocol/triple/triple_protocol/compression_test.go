// // Copyright 2021-2023 Buf Technologies, Inc.
// //
// // Licensed under the Apache License, Version 2.0 (the "License");
// // you may not use this file except in compliance with the License.
// // You may obtain a copy of the License at
// //
// //      http://www.apache.org/licenses/LICENSE-2.0
// //
// // Unless required by applicable law or agreed to in writing, software
// // distributed under the License is distributed on an "AS IS" BASIS,
// // WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// // See the License for the specific language governing permissions and
// // limitations under the License.
package triple_protocol

//
//import (
//	"context"
//	"net/http"
//	"net/http/httptest"
//	"testing"
//
//	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/assert"
//	"google.golang.org/protobuf/types/known/emptypb"
//)
//
//func TestAcceptEncodingOrdering(t *testing.T) {
//	t.Parallel()
//	const (
//		compressionBrotli = "br"
//		expect            = compressionGzip + "," + compressionBrotli
//	)
//
//	withFakeBrotli, ok := withGzip().(*compressionOption)
//	assert.True(t, ok)
//	withFakeBrotli.Name = compressionBrotli
//
//	var called bool
//	verify := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//		got := r.Header.Get(connectUnaryHeaderAcceptCompression)
//		assert.Equal(t, got, expect)
//		w.WriteHeader(http.StatusOK)
//		called = true
//	})
//	server := httptest.NewServer(verify)
//	defer server.Close()
//
//	client := NewClient[emptypb.Empty, emptypb.Empty](
//		server.Client(),
//		server.URL,
//		withFakeBrotli,
//		withGzip(),
//	)
//	_, _ = client.CallUnary(context.Background(), NewRequest(&emptypb.Empty{}))
//	assert.True(t, called)
//}
//
//func TestClientCompressionOptionTest(t *testing.T) {
//	t.Parallel()
//	const testURL = "http://foo.bar.com/service/method"
//
//	checkPools := func(t *testing.T, config *clientConfig) {
//		t.Helper()
//		assert.Equal(t, len(config.CompressionNames), len(config.CompressionPools))
//		for _, name := range config.CompressionNames {
//			pool := config.CompressionPools[name]
//			assert.NotNil(t, pool)
//		}
//	}
//	dummyDecompressCtor := func() Decompressor { return nil }
//	dummyCompressCtor := func() Compressor { return nil }
//
//	t.Run("defaults", func(t *testing.T) {
//		t.Parallel()
//		config, err := newClientConfig(testURL, nil)
//		assert.Nil(t, err)
//		assert.Equal(t, config.CompressionNames, []string{compressionGzip})
//		checkPools(t, config)
//	})
//	t.Run("WithAcceptCompression", func(t *testing.T) {
//		t.Parallel()
//		opts := []ClientOption{WithAcceptCompression("foo", dummyDecompressCtor, dummyCompressCtor)}
//		config, err := newClientConfig(testURL, opts)
//		assert.Nil(t, err)
//		assert.Equal(t, config.CompressionNames, []string{compressionGzip, "foo"})
//		checkPools(t, config)
//	})
//	t.Run("WithAcceptCompression-empty-name-noop", func(t *testing.T) {
//		t.Parallel()
//		opts := []ClientOption{WithAcceptCompression("", dummyDecompressCtor, dummyCompressCtor)}
//		config, err := newClientConfig(testURL, opts)
//		assert.Nil(t, err)
//		assert.Equal(t, config.CompressionNames, []string{compressionGzip})
//		checkPools(t, config)
//	})
//	t.Run("WithAcceptCompression-nil-ctors-noop", func(t *testing.T) {
//		t.Parallel()
//		opts := []ClientOption{WithAcceptCompression("foo", nil, nil)}
//		config, err := newClientConfig(testURL, opts)
//		assert.Nil(t, err)
//		assert.Equal(t, config.CompressionNames, []string{compressionGzip})
//		checkPools(t, config)
//	})
//	t.Run("WithAcceptCompression-nil-ctors-unregisters", func(t *testing.T) {
//		t.Parallel()
//		opts := []ClientOption{WithAcceptCompression("gzip", nil, nil)}
//		config, err := newClientConfig(testURL, opts)
//		assert.Nil(t, err)
//		assert.Equal(t, config.CompressionNames, nil)
//		checkPools(t, config)
//	})
//}
//
//func TestHandlerCompressionOptionTest(t *testing.T) {
//	t.Parallel()
//	const testProc = "/service/method"
//
//	checkPools := func(t *testing.T, config *handlerConfig) {
//		t.Helper()
//		assert.Equal(t, len(config.CompressionNames), len(config.CompressionPools))
//		for _, name := range config.CompressionNames {
//			pool := config.CompressionPools[name]
//			assert.NotNil(t, pool)
//		}
//	}
//	dummyDecompressCtor := func() Decompressor { return nil }
//	dummyCompressCtor := func() Compressor { return nil }
//
//	t.Run("defaults", func(t *testing.T) {
//		t.Parallel()
//		config := newHandlerConfig(testProc, nil)
//		assert.Equal(t, config.CompressionNames, []string{compressionGzip})
//		checkPools(t, config)
//	})
//	t.Run("WithCompression", func(t *testing.T) {
//		t.Parallel()
//		opts := []HandlerOption{WithCompression("foo", dummyDecompressCtor, dummyCompressCtor)}
//		config := newHandlerConfig(testProc, opts)
//		assert.Equal(t, config.CompressionNames, []string{compressionGzip, "foo"})
//		checkPools(t, config)
//	})
//	t.Run("WithCompression-empty-name-noop", func(t *testing.T) {
//		t.Parallel()
//		opts := []HandlerOption{WithCompression("", dummyDecompressCtor, dummyCompressCtor)}
//		config := newHandlerConfig(testProc, opts)
//		assert.Equal(t, config.CompressionNames, []string{compressionGzip})
//		checkPools(t, config)
//	})
//	t.Run("WithCompression-nil-ctors-noop", func(t *testing.T) {
//		t.Parallel()
//		opts := []HandlerOption{WithCompression("foo", nil, nil)}
//		config := newHandlerConfig(testProc, opts)
//		assert.Equal(t, config.CompressionNames, []string{compressionGzip})
//		checkPools(t, config)
//	})
//	t.Run("WithCompression-nil-ctors-unregisters", func(t *testing.T) {
//		t.Parallel()
//		opts := []HandlerOption{WithCompression("gzip", nil, nil)}
//		config := newHandlerConfig(testProc, opts)
//		assert.Equal(t, config.CompressionNames, nil)
//		checkPools(t, config)
//	})
//}
