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
	"net/http"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	triple "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
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

	require.Eventually(t, func() bool {
		conn, dialErr := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if dialErr != nil {
			return false
		}
		_ = conn.Close()
		return true
	}, 3*time.Second, 20*time.Millisecond)

	require.NoError(t, srv.Stop())
	select {
	case err := <-errCh:
		require.ErrorIs(t, err, http.ErrServerClosed)
	case <-time.After(5 * time.Second):
		require.FailNow(t, "server did not exit within 5s")
	}
}

// freeAddr returns a free TCP address on loopback for the test server.
func freeAddr(t *testing.T) string {
	t.Helper()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	addr := l.Addr().String()
	require.NoError(t, l.Close())
	return addr
}
