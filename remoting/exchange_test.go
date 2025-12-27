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

func TestSequenceID(t *testing.T) {
	id1, id2 := SequenceID(), SequenceID()
	assert.Equal(t, int64(2), id2-id1)
	assert.Equal(t, int64(0), id2%2)
}

func TestSequenceIDConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	ids := make(chan int64, 50)
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); ids <- SequenceID() }()
	}
	wg.Wait()
	close(ids)

	idSet := make(map[int64]bool)
	for id := range ids {
		assert.False(t, idSet[id])
		idSet[id] = true
	}
}

func TestNewRequest(t *testing.T) {
	req := NewRequest("2.0.2")
	assert.NotNil(t, req)
	assert.Equal(t, "2.0.2", req.Version)
	assert.NotEqual(t, int64(0), req.ID)
}

func TestNewResponse(t *testing.T) {
	resp := NewResponse(123, "2.0.2")
	assert.Equal(t, int64(123), resp.ID)
	assert.Equal(t, "2.0.2", resp.Version)
}

func TestResponseIsHeartbeat(t *testing.T) {
	assert.True(t, (&Response{Event: true, Result: nil}).IsHeartbeat())
	assert.False(t, (&Response{Event: true, Result: "data"}).IsHeartbeat())
	assert.False(t, (&Response{Event: false, Result: nil}).IsHeartbeat())
}

func TestResponseString(t *testing.T) {
	resp := &Response{ID: 123, Version: "2.0.2", Error: errors.New("err")}
	assert.Contains(t, resp.String(), "123")
	assert.Contains(t, resp.String(), "err")
}

func TestNewPendingResponse(t *testing.T) {
	pr := NewPendingResponse(100)
	assert.NotNil(t, pr)
	assert.Equal(t, int64(100), pr.seq)
	assert.NotNil(t, pr.Done)
}

func TestPendingResponseSetResponse(t *testing.T) {
	pr := NewPendingResponse(1)
	resp := &Response{ID: 1, Result: "result"}
	pr.SetResponse(resp)
	assert.Equal(t, resp, pr.response)
}

func TestPendingResponseGetCallResponse(t *testing.T) {
	pr := NewPendingResponse(1)
	pr.Err = errors.New("error")
	acr := pr.GetCallResponse().(AsyncCallbackResponse)
	assert.Equal(t, pr.Err, acr.Cause)
}

func TestAddGetRemovePendingResponse(t *testing.T) {
	pr := NewPendingResponse(999)
	AddPendingResponse(pr)
	assert.Equal(t, pr, GetPendingResponse(SequenceType(999)))
	assert.Equal(t, pr, removePendingResponse(SequenceType(999)))
	assert.Nil(t, removePendingResponse(SequenceType(999)))
}

func TestResponseHandle(t *testing.T) {
	t.Run("with callback", func(t *testing.T) {
		pr := NewPendingResponse(777)
		called := false
		pr.Callback = func(response common.CallbackResponse) { called = true }
		AddPendingResponse(pr)
		(&Response{ID: 777}).Handle()
		assert.True(t, called)
	})

	t.Run("without callback", func(t *testing.T) {
		pr := NewPendingResponse(666)
		AddPendingResponse(pr)
		(&Response{ID: 666, Error: errors.New("err")}).Handle()
		select {
		case <-pr.Done:
		case <-time.After(100 * time.Millisecond):
			t.Fatal("Done should be closed")
		}
		assert.Error(t, pr.Err)
	})
}
