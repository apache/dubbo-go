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
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

// mockHTTPClient implements HTTPClient for testing
type mockHTTPClient struct {
	response *http.Response
	err      error
	doFunc   func(*http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	if m.doFunc != nil {
		return m.doFunc(req)
	}
	return m.response, m.err
}

// TestNewDuplexHTTPCall tests newDuplexHTTPCall function
func TestNewDuplexHTTPCall(t *testing.T) {
	t.Parallel()

	t.Run("BasicCreation", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := &mockHTTPClient{}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}
		header := http.Header{"Content-Type": []string{"application/proto"}}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, header)

		assert.NotNil(t, call)
		assert.NotNil(t, call.request)
		assert.Equal(t, call.request.Method, http.MethodPost)
		assert.Equal(t, call.request.URL.String(), testURL.String())
		assert.Equal(t, call.request.Header.Get("Content-Type"), "application/proto")
	})

	t.Run("WithUserInfo", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := &mockHTTPClient{}
		testURL, _ := url.Parse("http://user:pass@example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)
		assert.NotNil(t, call)
		assert.NotNil(t, call.request.URL.User)
	})
}

// TestDuplexHTTPCallHeader tests Header method
func TestDuplexHTTPCallHeader(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &mockHTTPClient{}
	testURL, _ := url.Parse("http://example.com/service/method")
	spec := Spec{StreamType: StreamTypeUnary}
	header := http.Header{"X-Custom": []string{"value"}}

	call := newDuplexHTTPCall(ctx, client, testURL, spec, header)

	assert.Equal(t, call.Header().Get("X-Custom"), "value")

	// Modify header
	call.Header().Set("X-Another", "another-value")
	assert.Equal(t, call.Header().Get("X-Another"), "another-value")
}

// TestDuplexHTTPCallTrailer tests Trailer method
func TestDuplexHTTPCallTrailer(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &mockHTTPClient{}
	testURL, _ := url.Parse("http://example.com/service/method")
	spec := Spec{StreamType: StreamTypeUnary}

	call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)

	trailer := call.Trailer()
	assert.Nil(t, trailer)
}

// TestDuplexHTTPCallURL tests URL method
func TestDuplexHTTPCallURL(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &mockHTTPClient{}
	testURL, _ := url.Parse("http://example.com/service/method")
	spec := Spec{StreamType: StreamTypeUnary}

	call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)

	assert.Equal(t, call.URL().String(), testURL.String())
}

// TestDuplexHTTPCallSetMethod tests SetMethod method
func TestDuplexHTTPCallSetMethod(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &mockHTTPClient{}
	testURL, _ := url.Parse("http://example.com/service/method")
	spec := Spec{StreamType: StreamTypeUnary}

	call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)

	assert.Equal(t, call.request.Method, http.MethodPost)

	call.SetMethod(http.MethodGet)
	assert.Equal(t, call.request.Method, http.MethodGet)
}

// TestDuplexHTTPCallSetError tests SetError method
func TestDuplexHTTPCallSetError(t *testing.T) {
	t.Parallel()

	t.Run("SetOnce", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := &mockHTTPClient{}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)

		testErr := errors.New("test error")
		call.SetError(testErr)

		err := call.getError()
		assert.NotNil(t, err)
	})

	t.Run("SetMultipleTimes", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := &mockHTTPClient{}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)

		firstErr := errors.New("first error")
		secondErr := errors.New("second error")

		call.SetError(firstErr)
		call.SetError(secondErr)

		// Should keep first error
		err := call.getError()
		assert.NotNil(t, err)
		assert.True(t, strings.Contains(err.Error(), "first"))
	})

	t.Run("WithContextError", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := &mockHTTPClient{}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)

		call.SetError(context.Canceled)

		err := call.getError()
		assert.NotNil(t, err)
	})
}

// TestDuplexHTTPCallSetValidateResponse tests SetValidateResponse method
func TestDuplexHTTPCallSetValidateResponse(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &mockHTTPClient{}
	testURL, _ := url.Parse("http://example.com/service/method")
	spec := Spec{StreamType: StreamTypeUnary}

	call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)

	call.SetValidateResponse(func(r *http.Response) *Error {
		return nil
	})

	assert.NotNil(t, call.validateResponse)
}

// TestDuplexHTTPCallMakeRequest tests makeRequest behavior
func TestDuplexHTTPCallMakeRequest(t *testing.T) {
	t.Parallel()

	t.Run("ValidationError", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		response := &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}
		client := &mockHTTPClient{response: response}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)
		call.SetValidateResponse(func(r *http.Response) *Error {
			return NewError(CodeInvalidArgument, errors.New("validation failed"))
		})

		call.ensureRequestMade()
		call.BlockUntilResponseReady()

		err := call.getError()
		assert.NotNil(t, err)
	})

	t.Run("BidiStreamWithHTTP1", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		response := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
			ProtoMajor: 1,
			ProtoMinor: 1,
		}
		client := &mockHTTPClient{response: response}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeBidi}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)
		call.SetValidateResponse(func(r *http.Response) *Error { return nil })

		call.ensureRequestMade()
		call.BlockUntilResponseReady()

		err := call.getError()
		assert.NotNil(t, err)
		tripleErr, ok := err.(*Error)
		assert.True(t, ok)
		assert.Equal(t, tripleErr.Code(), CodeUnimplemented)
	})

	t.Run("HTTPClientError", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := &mockHTTPClient{err: errors.New("connection refused")}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)
		call.SetValidateResponse(func(r *http.Response) *Error { return nil })

		call.ensureRequestMade()
		call.BlockUntilResponseReady()

		err := call.getError()
		assert.NotNil(t, err)
	})
}

// TestDuplexHTTPCallResponseMethods tests response-related methods
func TestDuplexHTTPCallResponseMethods(t *testing.T) {
	t.Parallel()

	t.Run("ResponseStatusCode_Success", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		response := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
		}
		client := &mockHTTPClient{response: response}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)
		call.SetValidateResponse(func(r *http.Response) *Error { return nil })

		call.ensureRequestMade()
		call.BlockUntilResponseReady()

		code, err := call.ResponseStatusCode()
		assert.Nil(t, err)
		assert.Equal(t, code, http.StatusOK)
	})

	t.Run("ResponseStatusCode_NilResponse", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := &mockHTTPClient{err: errors.New("error")}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)
		call.SetValidateResponse(func(r *http.Response) *Error { return nil })

		call.ensureRequestMade()
		call.BlockUntilResponseReady()

		_, err := call.ResponseStatusCode()
		assert.NotNil(t, err)
	})

	t.Run("ResponseHeader_WithResponse", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		response := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{"X-Custom": []string{"value"}},
		}
		client := &mockHTTPClient{response: response}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)
		call.SetValidateResponse(func(r *http.Response) *Error { return nil })

		call.ensureRequestMade()
		call.BlockUntilResponseReady()

		header := call.ResponseHeader()
		assert.Equal(t, header.Get("X-Custom"), "value")
	})

	t.Run("ResponseHeader_NilResponse", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := &mockHTTPClient{err: errors.New("error")}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)
		call.SetValidateResponse(func(r *http.Response) *Error { return nil })

		call.ensureRequestMade()
		call.BlockUntilResponseReady()

		header := call.ResponseHeader()
		assert.NotNil(t, header)
		assert.Equal(t, len(header), 0)
	})

	t.Run("ResponseTrailer_WithResponse", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		response := &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader("")),
			Header:     http.Header{},
			Trailer:    http.Header{"X-Trailer": []string{"trailer-value"}},
		}
		client := &mockHTTPClient{response: response}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)
		call.SetValidateResponse(func(r *http.Response) *Error { return nil })

		call.ensureRequestMade()
		call.BlockUntilResponseReady()

		trailer := call.ResponseTrailer()
		assert.Equal(t, trailer.Get("X-Trailer"), "trailer-value")
	})

	t.Run("ResponseTrailer_NilResponse", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		client := &mockHTTPClient{err: errors.New("error")}
		testURL, _ := url.Parse("http://example.com/service/method")
		spec := Spec{StreamType: StreamTypeUnary}

		call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)
		call.SetValidateResponse(func(r *http.Response) *Error { return nil })

		call.ensureRequestMade()
		call.BlockUntilResponseReady()

		trailer := call.ResponseTrailer()
		assert.NotNil(t, trailer)
		assert.Equal(t, len(trailer), 0)
	})
}

// TestCloneURL tests cloneURL function
func TestCloneURL(t *testing.T) {
	t.Parallel()

	t.Run("NilURL", func(t *testing.T) {
		t.Parallel()
		result := cloneURL(nil)
		assert.Nil(t, result)
	})

	t.Run("BasicURL", func(t *testing.T) {
		t.Parallel()
		original, _ := url.Parse("http://example.com/path?query=value")
		cloned := cloneURL(original)

		assert.NotNil(t, cloned)
		assert.Equal(t, cloned.String(), original.String())

		// Modify cloned URL should not affect original
		cloned.Path = "/different"
		assert.NotEqual(t, cloned.Path, original.Path)
	})

	t.Run("URLWithUserInfo", func(t *testing.T) {
		t.Parallel()
		original, _ := url.Parse("http://user:pass@example.com/path")
		cloned := cloneURL(original)

		assert.NotNil(t, cloned)
		assert.NotNil(t, cloned.User)
		assert.Equal(t, cloned.User.Username(), "user")

		password, _ := cloned.User.Password()
		assert.Equal(t, password, "pass")
	})
}

// TestDuplexHTTPCallEnsureRequestMadeOnce tests that request is made only once
func TestDuplexHTTPCallEnsureRequestMadeOnce(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	callCount := 0
	client := &mockHTTPClient{
		doFunc: func(req *http.Request) (*http.Response, error) {
			callCount++
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader("")),
				Header:     http.Header{},
			}, nil
		},
	}
	testURL, _ := url.Parse("http://example.com/service/method")
	spec := Spec{StreamType: StreamTypeUnary}

	call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)
	call.SetValidateResponse(func(r *http.Response) *Error { return nil })

	// Call multiple times
	call.ensureRequestMade()
	call.ensureRequestMade()
	call.ensureRequestMade()

	call.BlockUntilResponseReady()

	// Should only be called once
	assert.Equal(t, callCount, 1)
}

// TestDuplexHTTPCallConcurrentSetError tests concurrent SetError calls
func TestDuplexHTTPCallConcurrentSetError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	client := &mockHTTPClient{}
	testURL, _ := url.Parse("http://example.com/service/method")
	spec := Spec{StreamType: StreamTypeUnary}

	call := newDuplexHTTPCall(ctx, client, testURL, spec, nil)

	// Concurrent SetError calls
	done := make(chan struct{})
	for i := 0; i < 10; i++ {
		go func() {
			call.SetError(errors.New("error"))
			done <- struct{}{}
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should not panic and error should be set
	err := call.getError()
	assert.NotNil(t, err)
}
