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

package remoting

import (
	"errors"
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// ============================================
// SequenceID Tests
// ============================================
func TestSequenceID(t *testing.T) {
	t.Run("sequence increases by 2", func(t *testing.T) {
		id1 := SequenceID()
		id2 := SequenceID()
		assert.Equal(t, int64(2), id2-id1)
	})

	t.Run("sequence is always even", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			id := SequenceID()
			assert.Equal(t, int64(0), id%2)
		}
	})
}

func TestSequenceIDConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	ids := make(chan int64, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ids <- SequenceID()
		}()
	}

	wg.Wait()
	close(ids)

	// Collect all IDs
	idSet := make(map[int64]bool)
	for id := range ids {
		assert.False(t, idSet[id], "Duplicate ID found: %d", id)
		idSet[id] = true
	}
	assert.Equal(t, 100, len(idSet))
}

// ============================================
// Request Tests
// ============================================

func TestNewRequest(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		validate func(*testing.T, *Request)
	}{
		{
			name:    "standard version",
			version: "2.0.2",
			validate: func(t *testing.T, req *Request) {
				assert.Equal(t, "2.0.2", req.Version)
				assert.NotEqual(t, int64(0), req.ID)
			},
		},
		{
			name:    "empty version",
			version: "",
			validate: func(t *testing.T, req *Request) {
				assert.Equal(t, "", req.Version)
			},
		},
		{
			name:    "custom version",
			version: "3.0.0",
			validate: func(t *testing.T, req *Request) {
				assert.Equal(t, "3.0.0", req.Version)
			},
		},
		{
			name:    "version with suffix",
			version: "2.0.2-SNAPSHOT",
			validate: func(t *testing.T, req *Request) {
				assert.Equal(t, "2.0.2-SNAPSHOT", req.Version)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := NewRequest(tt.version)
			assert.NotNil(t, req)
			tt.validate(t, req)
		})
	}
}

func TestRequestFields(t *testing.T) {
	tests := []struct {
		name     string
		request  *Request
		validate func(*testing.T, *Request)
	}{
		{
			name: "full request",
			request: &Request{
				ID:       100,
				Version:  "2.0.2",
				SerialID: 1,
				Data:     "test-data",
				TwoWay:   true,
				Event:    false,
			},
			validate: func(t *testing.T, req *Request) {
				assert.Equal(t, int64(100), req.ID)
				assert.Equal(t, "2.0.2", req.Version)
				assert.Equal(t, byte(1), req.SerialID)
				assert.Equal(t, "test-data", req.Data)
				assert.True(t, req.TwoWay)
				assert.False(t, req.Event)
			},
		},
		{
			name: "event request",
			request: &Request{
				ID:     200,
				Event:  true,
				TwoWay: false,
			},
			validate: func(t *testing.T, req *Request) {
				assert.True(t, req.Event)
				assert.False(t, req.TwoWay)
			},
		},
		{
			name: "request with complex data",
			request: &Request{
				ID:   300,
				Data: map[string]any{"method": "invoke", "args": []string{"a", "b"}},
			},
			validate: func(t *testing.T, req *Request) {
				data, ok := req.Data.(map[string]any)
				assert.True(t, ok)
				assert.Equal(t, "invoke", data["method"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.request)
		})
	}
}

// ============================================
// Response Tests
// ============================================

func TestNewResponse(t *testing.T) {
	tests := []struct {
		name    string
		id      int64
		version string
	}{
		{"standard response", 1, "2.0.2"},
		{"zero id", 0, "2.0.2"},
		{"negative id", -1, "2.0.2"},
		{"empty version", 1, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := NewResponse(tt.id, tt.version)
			assert.NotNil(t, resp)
			assert.Equal(t, tt.id, resp.ID)
			assert.Equal(t, tt.version, resp.Version)
		})
	}
}

func TestResponseIsHeartbeat(t *testing.T) {
	tests := []struct {
		name     string
		event    bool
		result   any
		expected bool
	}{
		{"heartbeat response", true, nil, true},
		{"non-heartbeat with event", true, "data", false},
		{"non-heartbeat without event", false, nil, false},
		{"non-heartbeat with data", false, "data", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &Response{
				Event:  tt.event,
				Result: tt.result,
			}
			assert.Equal(t, tt.expected, resp.IsHeartbeat())
		})
	}
}

func TestResponseString(t *testing.T) {
	resp := &Response{
		ID:       123,
		Version:  "2.0.2",
		SerialID: 1,
		Status:   20,
		Event:    false,
		Error:    nil,
		Result:   "test-result",
	}

	str := resp.String()
	assert.Contains(t, str, "123")
	assert.Contains(t, str, "2.0.2")
	assert.Contains(t, str, "remoting.Response")
}

func TestResponseStringWithError(t *testing.T) {
	resp := &Response{
		ID:      456,
		Version: "2.0.2",
		Error:   errors.New("test error"),
	}

	str := resp.String()
	assert.Contains(t, str, "456")
	assert.Contains(t, str, "test error")
}

// ============================================
// Options Tests
// ============================================

func TestOptions(t *testing.T) {
	tests := []struct {
		name           string
		connectTimeout time.Duration
	}{
		{"default timeout", 0},
		{"short timeout", 100 * time.Millisecond},
		{"long timeout", 30 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := Options{
				ConnectTimeout: tt.connectTimeout,
			}
			assert.Equal(t, tt.connectTimeout, opts.ConnectTimeout)
		})
	}
}

// ============================================
// AsyncCallbackResponse Tests
// ============================================

func TestAsyncCallbackResponse(t *testing.T) {
	now := time.Now()
	acr := AsyncCallbackResponse{
		Opts:      Options{ConnectTimeout: 5 * time.Second},
		Cause:     errors.New("test cause"),
		Start:     now,
		ReadStart: now.Add(100 * time.Millisecond),
		Reply:     "test-reply",
	}

	assert.Equal(t, 5*time.Second, acr.Opts.ConnectTimeout)
	assert.NotNil(t, acr.Cause)
	assert.Equal(t, "test cause", acr.Cause.Error())
	assert.Equal(t, now, acr.Start)
	assert.Equal(t, "test-reply", acr.Reply)
}

// ============================================
// PendingResponse Tests
// ============================================

func TestNewPendingResponse(t *testing.T) {
	tests := []struct {
		name string
		id   int64
	}{
		{"positive id", 100},
		{"zero id", 0},
		{"negative id", -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := NewPendingResponse(tt.id)
			assert.NotNil(t, pr)
			assert.Equal(t, tt.id, pr.seq)
			assert.NotNil(t, pr.Done)
			assert.NotNil(t, pr.response)
		})
	}
}

func TestPendingResponseSetResponse(t *testing.T) {
	pr := NewPendingResponse(1)
	resp := &Response{
		ID:     1,
		Result: "new-result",
	}

	pr.SetResponse(resp)
	assert.Equal(t, resp, pr.response)
}

func TestPendingResponseGetCallResponse(t *testing.T) {
	pr := NewPendingResponse(1)
	pr.Err = errors.New("test error")
	pr.response = &Response{Result: "test-result"}

	callResp := pr.GetCallResponse()
	acr, ok := callResp.(AsyncCallbackResponse)
	assert.True(t, ok)
	assert.Equal(t, pr.Err, acr.Cause)
}

// ============================================
// PendingResponse Storage Tests
// ============================================

func TestAddAndGetPendingResponse(t *testing.T) {
	pr := NewPendingResponse(999)
	AddPendingResponse(pr)

	retrieved := GetPendingResponse(SequenceType(999))
	assert.NotNil(t, retrieved)
	assert.Equal(t, pr, retrieved)
}

func TestGetPendingResponseNotFound(t *testing.T) {
	result := GetPendingResponse(SequenceType(-99999))
	assert.Nil(t, result)
}

func TestRemovePendingResponse(t *testing.T) {
	pr := NewPendingResponse(888)
	AddPendingResponse(pr)

	// First removal should succeed
	removed := removePendingResponse(SequenceType(888))
	assert.NotNil(t, removed)
	assert.Equal(t, pr, removed)

	// Second removal should return nil
	removed2 := removePendingResponse(SequenceType(888))
	assert.Nil(t, removed2)
}

// ============================================
// Response Handle Tests
// ============================================

func TestResponseHandleWithCallback(t *testing.T) {
	pr := NewPendingResponse(777)
	callbackCalled := false
	pr.Callback = func(response common.CallbackResponse) {
		callbackCalled = true
	}
	AddPendingResponse(pr)

	resp := &Response{
		ID:     777,
		Result: "callback-result",
	}
	resp.Handle()

	assert.True(t, callbackCalled)
}

func TestResponseHandleWithoutCallback(t *testing.T) {
	pr := NewPendingResponse(666)
	AddPendingResponse(pr)

	resp := &Response{
		ID:     666,
		Result: "no-callback-result",
		Error:  errors.New("test error"),
	}
	resp.Handle()

	// Done channel should be closed
	select {
	case <-pr.Done:
		// Expected
	default:
		t.Fatal("Done channel should be closed")
	}
	assert.Equal(t, resp.Error, pr.Err)
}

func TestResponseHandleNotFound(t *testing.T) {
	resp := &Response{
		ID: -12345, // Non-existent ID
	}
	// Should not panic
	resp.Handle()
}

// ============================================
// Concurrent PendingResponse Tests
// ============================================

func TestPendingResponseConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup

	// Concurrent add
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			pr := NewPendingResponse(int64(10000 + idx))
			AddPendingResponse(pr)
		}(i)
	}

	wg.Wait()

	// Concurrent get and remove
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_ = GetPendingResponse(SequenceType(10000 + idx))
			_ = removePendingResponse(SequenceType(10000 + idx))
		}(i)
	}

	wg.Wait()
}

// ============================================
// Edge Cases Tests
// ============================================

func TestRequestEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		request  *Request
		validate func(*testing.T, *Request)
	}{
		{
			name: "nil data",
			request: &Request{
				ID:      1,
				Version: "2.0.2",
				Data:    nil,
			},
			validate: func(t *testing.T, req *Request) {
				assert.Nil(t, req.Data)
			},
		},
		{
			name: "zero id",
			request: &Request{
				ID:      0,
				Version: "2.0.2",
			},
			validate: func(t *testing.T, req *Request) {
				assert.Equal(t, int64(0), req.ID)
			},
		},
		{
			name: "negative id",
			request: &Request{
				ID: -1,
			},
			validate: func(t *testing.T, req *Request) {
				assert.Equal(t, int64(-1), req.ID)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.request)
		})
	}
}

func TestResponseEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		response *Response
		validate func(*testing.T, *Response)
	}{
		{
			name: "nil error",
			response: &Response{
				ID:    1,
				Error: nil,
			},
			validate: func(t *testing.T, resp *Response) {
				assert.Nil(t, resp.Error)
				str := resp.String()
				assert.Contains(t, str, "<nil>")
			},
		},
		{
			name: "nil result",
			response: &Response{
				ID:     2,
				Result: nil,
			},
			validate: func(t *testing.T, resp *Response) {
				assert.Nil(t, resp.Result)
			},
		},
		{
			name: "all status values",
			response: &Response{
				ID:     3,
				Status: 255,
			},
			validate: func(t *testing.T, resp *Response) {
				assert.Equal(t, uint8(255), resp.Status)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.response)
		})
	}
}

func TestPendingResponseEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *PendingResponse
		validate func(*testing.T, *PendingResponse)
	}{
		{
			name: "nil callback",
			setup: func() *PendingResponse {
				pr := NewPendingResponse(555)
				pr.Callback = nil
				return pr
			},
			validate: func(t *testing.T, pr *PendingResponse) {
				AddPendingResponse(pr)
				resp := &Response{ID: 555}
				resp.Handle()
				select {
				case <-pr.Done:
					// Expected
				case <-time.After(100 * time.Millisecond):
					t.Fatal("Done channel should be closed")
				}
			},
		},
		{
			name: "with reply",
			setup: func() *PendingResponse {
				pr := NewPendingResponse(556)
				pr.Reply = "expected-reply"
				return pr
			},
			validate: func(t *testing.T, pr *PendingResponse) {
				assert.Equal(t, "expected-reply", pr.Reply)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := tt.setup()
			tt.validate(t, pr)
		})
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkSequenceID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		SequenceID()
	}
}

func BenchmarkNewRequest(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewRequest("2.0.2")
	}
}

func BenchmarkNewResponse(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewResponse(1, "2.0.2")
	}
}

func BenchmarkNewPendingResponse(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewPendingResponse(int64(i))
	}
}

func BenchmarkAddPendingResponse(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pr := NewPendingResponse(int64(i + 100000))
		AddPendingResponse(pr)
	}
}

func BenchmarkGetPendingResponse(b *testing.B) {
	pr := NewPendingResponse(99999)
	AddPendingResponse(pr)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		GetPendingResponse(SequenceType(99999))
	}
}

func BenchmarkResponseString(b *testing.B) {
	resp := &Response{
		ID:      123,
		Version: "2.0.2",
		Status:  20,
		Result:  "test-result",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = resp.String()
	}
}
