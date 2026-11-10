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

package triple_protocol_test

import (
	"net"
	"testing"
	"time"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	triple "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol/internal/assert"
)

// TestServer_Run_KeepsTwoArgumentSignature verifies that the exported Run
// keeps its pre-3640 two-argument signature: the exact call shape a
// pre-3640 consumer writes still compiles and serves, then stops cleanly.
func TestServer_Run_KeepsTwoArgumentSignature(t *testing.T) {
	addr := freeAddr(t)
	srv := triple.NewServer(addr, nil)
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Run(constant.CallHTTP2, nil)
	}()

	// Wait until the listener is up so the stop below closes a serving
	// server instead of racing the startup.
	waitForListener(t, addr)

	assert.Nil(t, srv.Stop())
	select {
	case err := <-errCh:
		// A clean Stop is the normal end of the single-protocol path: the
		// shutdown filter suppresses http.ErrServerClosed, so Run returns nil.
		assert.Nil(t, err)
	case <-time.After(5 * time.Second):
		t.Fatalf("server did not exit within 5s")
	}
}

// freeAddr returns a free TCP address on loopback for the test server.
func freeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	assert.Nil(t, err)
	addr := l.Addr().String()
	assert.Nil(t, l.Close())
	return addr
}

// waitForListener polls until the address accepts TCP connections or the
// deadline expires.
func waitForListener(t *testing.T, addr string) {
	t.Helper()

	deadline := time.Now().Add(3 * time.Second)
	for {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("server did not accept connections within 3s: %v", err)
		}
		time.Sleep(20 * time.Millisecond)
	}
}
