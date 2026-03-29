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

package triple

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func TestNewTripleProtocol(t *testing.T) {
	tp := NewTripleProtocol()

	assert.NotNil(t, tp)
	assert.NotNil(t, tp.serverMap)
	assert.Empty(t, tp.serverMap)
}

func TestGetProtocol(t *testing.T) {
	// reset singleton for test isolation
	tripleProtocol = nil

	p1 := GetProtocol()
	assert.NotNil(t, p1)

	// should return same instance (singleton)
	p2 := GetProtocol()
	assert.Same(t, p1, p2)
}

func TestTripleProtocolRegistration(t *testing.T) {
	// verify protocol is registered via init()
	p := extension.GetProtocol(TRIPLE)
	assert.NotNil(t, p)
}

func TestTripleConstant(t *testing.T) {
	assert.Equal(t, "tri", TRIPLE)
}

func TestTripleGracefulShutdownCallbackRegistration(t *testing.T) {
	cb, ok := extension.LookupGracefulShutdownCallback(TRIPLE)
	assert.True(t, ok)
	assert.NotNil(t, cb)

	original := tripleProtocol
	tp := NewTripleProtocol()
	tp.serverMap["graceful-test"] = &Server{triServer: tri.NewServer("", nil)}
	tripleProtocol = tp
	t.Cleanup(func() {
		tripleProtocol = original
	})

	assert.NotPanics(t, func() {
		assert.NoError(t, cb(context.Background()))
	})
}

func TestTripleProtocol_Destroy_EmptyServerMap(t *testing.T) {
	tp := NewTripleProtocol()

	// should not panic when serverMap is empty
	assert.NotPanics(t, func() {
		tp.Destroy()
	})
}

func TestTripleProtocolDestroyDoesNotHoldServerLockWhileGracefulStopping(t *testing.T) {
	tp := NewTripleProtocol()
	tp.serverMap["127.0.0.1:20000"] = &Server{}

	originalStop := tripleServerGracefulStop
	t.Cleanup(func() {
		tripleServerGracefulStop = originalStop
	})

	entered := make(chan struct{})
	release := make(chan struct{})
	var stopCalls atomic.Int32
	tripleServerGracefulStop = func(server *Server) {
		stopCalls.Add(1)
		close(entered)
		<-release
	}

	done := make(chan struct{})
	go func() {
		tp.Destroy()
		close(done)
	}()

	select {
	case <-entered:
	case <-time.After(time.Second):
		t.Fatal("Destroy did not reach graceful stop")
	}

	lockAcquired := make(chan struct{})
	go func() {
		tp.serverLock.Lock()
		_ = tp.serverMap
		tp.serverLock.Unlock()
		close(lockAcquired)
	}()

	select {
	case <-lockAcquired:
	case <-time.After(time.Second):
		t.Fatal("serverLock remained held during graceful stop")
	}

	close(release)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Destroy did not finish")
	}

	assert.Equal(t, int32(1), stopCalls.Load())
	assert.Empty(t, tp.serverMap)
}

// Test isGenericCall checks if the generic parameter indicates a generic call
func Test_isGenericCall(t *testing.T) {
	tests := []struct {
		name     string
		generic  string
		expected bool
	}{
		// valid generic serialization types
		{"empty string", "", false},
		{"true", "true", true},
		{"TRUE", "TRUE", true},
		{"True", "True", true},
		{"gson", "gson", true},
		{"GSON", "GSON", true},
		{"Gson", "Gson", true},
		{"protobuf", "protobuf", true},
		{"PROTOBUF", "PROTOBUF", true},
		{"Protobuf", "Protobuf", true},
		{"protobuf-json", "protobuf-json", true},
		{"PROTOBUF-JSON", "PROTOBUF-JSON", true},
		{"Protobuf-Json", "Protobuf-Json", true},

		// invalid generic serialization types
		{"false", "false", false},
		{"random", "random", false},
		{"json", "json", false},
		{"xml", "xml", false},
		{"hessian", "hessian", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isGenericCall(tt.generic)
			assert.Equal(t, tt.expected, result)
		})
	}
}
