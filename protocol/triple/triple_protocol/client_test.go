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
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
	pingv1 "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/gen/proto/connect/ping/v1"
)

func TestNewClient_InvalidURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "invalid url format",
			url:     "://invalid",
			wantErr: true,
		},
		{
			name:    "valid http url",
			url:     "http://localhost:8080/test",
			wantErr: false,
		},
		{
			name:    "valid https url",
			url:     "https://localhost:8080/test",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			client := NewClient(http.DefaultClient, tt.url)
			if tt.wantErr {
				assert.NotNil(t, client.err)
			} else {
				assert.Nil(t, client.err)
			}
		})
	}
}

func TestNewClient_InvalidCompression(t *testing.T) {
	t.Parallel()
	client := NewClient(
		http.DefaultClient,
		"http://localhost:8080/test",
		WithSendCompression("invalid-compression"),
	)
	assert.NotNil(t, client.err)
}

func TestClient_CallUnary_WithError(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	client := &Client{err: initErr}

	req := NewRequest(&pingv1.PingRequest{})
	res := NewResponse(&pingv1.PingResponse{})
	err := client.CallUnary(context.Background(), req, res)
	assert.ErrorIs(t, err, initErr)
}

func TestClient_CallClientStream_WithError(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	client := &Client{err: initErr}

	stream, err := client.CallClientStream(context.Background())
	assert.ErrorIs(t, err, initErr)
	assert.NotNil(t, stream)
	assert.ErrorIs(t, stream.err, initErr)
}

func TestClient_CallServerStream_WithError(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	client := &Client{err: initErr}

	req := NewRequest(&pingv1.PingRequest{})
	stream, err := client.CallServerStream(context.Background(), req)
	assert.ErrorIs(t, err, initErr)
	assert.Nil(t, stream)
}

func TestClient_CallBidiStream_WithError(t *testing.T) {
	t.Parallel()
	initErr := errors.New("client init failure")
	client := &Client{err: initErr}

	stream, err := client.CallBidiStream(context.Background())
	assert.ErrorIs(t, err, initErr)
	assert.NotNil(t, stream)
	assert.ErrorIs(t, stream.err, initErr)
}

func TestParseRequestURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		rawURL  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid http url",
			rawURL:  "http://localhost:8080/test",
			wantErr: false,
		},
		{
			name:    "valid https url",
			rawURL:  "https://localhost:8080/test",
			wantErr: false,
		},
		{
			name:    "invalid url format",
			rawURL:  "://invalid",
			wantErr: true,
		},
		{
			name:    "empty url",
			rawURL:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			url, err := parseRequestURL(tt.rawURL)
			if tt.wantErr {
				assert.NotNil(t, err)
				if tt.errMsg != "" {
					assert.Match(t, err.Message(), tt.errMsg)
				}
			} else {
				assert.Nil(t, err)
				assert.NotNil(t, url)
			}
		})
	}
}

func TestApplyDefaultTimeout(t *testing.T) {
	t.Parallel()

	t.Run("no timeout configured", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		newCtx, flag, cancel := applyDefaultTimeout(ctx, 0)
		assert.False(t, flag)
		assert.Nil(t, cancel)
		_, hasDeadline := newCtx.Deadline()
		assert.False(t, hasDeadline)
	})

	t.Run("with timeout configured", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		timeout := 5 * time.Second
		newCtx, flag, cancel := applyDefaultTimeout(ctx, timeout)
		assert.True(t, flag)
		assert.NotNil(t, cancel)
		defer cancel()
		deadline, hasDeadline := newCtx.Deadline()
		assert.True(t, hasDeadline)
		assert.True(t, deadline.After(time.Now()))
	})

	t.Run("context already has deadline", func(t *testing.T) {
		t.Parallel()
		ctx, existingCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer existingCancel()
		newCtx, flag, cancel := applyDefaultTimeout(ctx, 5*time.Second)
		assert.False(t, flag)
		assert.Nil(t, cancel)
		// Should keep existing deadline
		_, hasDeadline := newCtx.Deadline()
		assert.True(t, hasDeadline)
	})

	t.Run("with TimeoutKey in context", func(t *testing.T) {
		t.Parallel()
		ctx := context.WithValue(context.Background(), TimeoutKey{}, "3s")
		newCtx, flag, cancel := applyDefaultTimeout(ctx, 0)
		assert.True(t, flag)
		assert.NotNil(t, cancel)
		defer cancel()
		deadline, hasDeadline := newCtx.Deadline()
		assert.True(t, hasDeadline)
		assert.True(t, deadline.After(time.Now()))
	})

	t.Run("with invalid TimeoutKey value", func(t *testing.T) {
		t.Parallel()
		ctx := context.WithValue(context.Background(), TimeoutKey{}, "invalid")
		newCtx, flag, cancel := applyDefaultTimeout(ctx, 0)
		assert.False(t, flag)
		assert.Nil(t, cancel)
		_, hasDeadline := newCtx.Deadline()
		assert.False(t, hasDeadline)
	})

	t.Run("with empty TimeoutKey value", func(t *testing.T) {
		t.Parallel()
		ctx := context.WithValue(context.Background(), TimeoutKey{}, "")
		newCtx, flag, cancel := applyDefaultTimeout(ctx, 0)
		assert.False(t, flag)
		assert.Nil(t, cancel)
		_, hasDeadline := newCtx.Deadline()
		assert.False(t, hasDeadline)
	})

	t.Run("with non-string TimeoutKey value", func(t *testing.T) {
		t.Parallel()
		ctx := context.WithValue(context.Background(), TimeoutKey{}, 123)
		newCtx, flag, cancel := applyDefaultTimeout(ctx, 0)
		assert.False(t, flag)
		assert.Nil(t, cancel)
		_, hasDeadline := newCtx.Deadline()
		assert.False(t, hasDeadline)
	})
}

func TestApplyGroupVersionHeaders(t *testing.T) {
	t.Parallel()

	t.Run("with group and version", func(t *testing.T) {
		t.Parallel()
		header := make(http.Header)
		cfg := &clientConfig{
			Group:   "test-group",
			Version: "1.0.0",
		}
		applyGroupVersionHeaders(header, cfg)
		assert.Equal(t, header.Get(tripleServiceGroup), "test-group")
		assert.Equal(t, header.Get(tripleServiceVersion), "1.0.0")
	})

	t.Run("with only group", func(t *testing.T) {
		t.Parallel()
		header := make(http.Header)
		cfg := &clientConfig{
			Group: "test-group",
		}
		applyGroupVersionHeaders(header, cfg)
		assert.Equal(t, header.Get(tripleServiceGroup), "test-group")
		assert.Equal(t, header.Get(tripleServiceVersion), "")
	})

	t.Run("with only version", func(t *testing.T) {
		t.Parallel()
		header := make(http.Header)
		cfg := &clientConfig{
			Version: "1.0.0",
		}
		applyGroupVersionHeaders(header, cfg)
		assert.Equal(t, header.Get(tripleServiceGroup), "")
		assert.Equal(t, header.Get(tripleServiceVersion), "1.0.0")
	})

	t.Run("with empty group and version", func(t *testing.T) {
		t.Parallel()
		header := make(http.Header)
		cfg := &clientConfig{}
		applyGroupVersionHeaders(header, cfg)
		assert.Equal(t, header.Get(tripleServiceGroup), "")
		assert.Equal(t, header.Get(tripleServiceVersion), "")
	})
}

func TestClientConfig_Validate(t *testing.T) {
	t.Parallel()

	t.Run("nil codec", func(t *testing.T) {
		t.Parallel()
		cfg := &clientConfig{
			Codec: nil,
		}
		err := cfg.validate()
		assert.NotNil(t, err)
		assert.Match(t, err.Message(), "no codec configured")
	})

	t.Run("empty codec name", func(t *testing.T) {
		t.Parallel()
		cfg := &clientConfig{
			Codec: &emptyNameCodec{},
		}
		err := cfg.validate()
		assert.NotNil(t, err)
		assert.Match(t, err.Message(), "no codec configured")
	})

	t.Run("unknown compression", func(t *testing.T) {
		t.Parallel()
		cfg := &clientConfig{
			Codec:                  &protoBinaryCodec{},
			RequestCompressionName: "unknown",
			CompressionPools:       make(map[string]*compressionPool),
		}
		err := cfg.validate()
		assert.NotNil(t, err)
		assert.Match(t, err.Message(), `unknown compression "unknown"`)
	})

	t.Run("valid config", func(t *testing.T) {
		t.Parallel()
		cfg := &clientConfig{
			Codec:                  &protoBinaryCodec{},
			RequestCompressionName: "",
			CompressionPools:       make(map[string]*compressionPool),
		}
		err := cfg.validate()
		assert.Nil(t, err)
	})

	t.Run("identity compression", func(t *testing.T) {
		t.Parallel()
		cfg := &clientConfig{
			Codec:                  &protoBinaryCodec{},
			RequestCompressionName: compressionIdentity,
			CompressionPools:       make(map[string]*compressionPool),
		}
		err := cfg.validate()
		assert.Nil(t, err)
	})
}

func TestClientConfig_Protobuf(t *testing.T) {
	t.Parallel()

	t.Run("codec is proto", func(t *testing.T) {
		t.Parallel()
		protoCodec := &protoBinaryCodec{}
		cfg := &clientConfig{
			Codec: protoCodec,
		}
		result := cfg.protobuf()
		assert.Equal(t, result, protoCodec)
	})

	t.Run("codec is not proto", func(t *testing.T) {
		t.Parallel()
		jsonCodec := &protoJSONCodec{}
		cfg := &clientConfig{
			Codec: jsonCodec,
		}
		result := cfg.protobuf()
		_, ok := result.(*protoBinaryCodec)
		assert.True(t, ok)
	})
}

func TestClientConfig_NewSpec(t *testing.T) {
	t.Parallel()

	cfg := &clientConfig{
		Procedure:        "/test.Service/Method",
		IdempotencyLevel: IdempotencyNoSideEffects,
	}

	tests := []struct {
		name       string
		streamType StreamType
	}{
		{"unary", StreamTypeUnary},
		{"client", StreamTypeClient},
		{"server", StreamTypeServer},
		{"bidi", StreamTypeBidi},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			spec := cfg.newSpec(tt.streamType)
			assert.Equal(t, spec.StreamType, tt.streamType)
			assert.Equal(t, spec.Procedure, "/test.Service/Method")
			assert.True(t, spec.IsClient)
			assert.Equal(t, spec.IdempotencyLevel, IdempotencyNoSideEffects)
		})
	}
}

// emptyNameCodec is a codec that returns empty name for testing
type emptyNameCodec struct{}

func (c *emptyNameCodec) Name() string                { return "" }
func (c *emptyNameCodec) Marshal(any) ([]byte, error) { return nil, nil }
func (c *emptyNameCodec) Unmarshal([]byte, any) error { return nil }
