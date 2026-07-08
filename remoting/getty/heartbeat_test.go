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

package getty

import (
	"testing"
	"time"
)

import (
	getty "github.com/apache/dubbo-getty"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// mockHeartbeatSession embeds getty.Session so it satisfies the interface, and
// overrides only WritePkg. It captures the ID of the heartbeat request that was
// written so the test can assert the pending response is cleaned up afterwards.
type mockHeartbeatSession struct {
	getty.Session
	writtenID int64
}

func (m *mockHeartbeatSession) WritePkg(pkg interface{}, _ time.Duration) (int, int, error) {
	if req, ok := pkg.(*remoting.Request); ok {
		m.writtenID = req.ID
	}
	// Return sendLen == 0 so heartbeat does not try to close the session.
	return 0, 0, nil
}

// TestHeartbeatTimeoutCleanup is a regression test for the map leak fix: when a
// heartbeat times out (no response arrives), the pending response registered for
// the heartbeat request ID must be removed from remoting's pendingResponses so the
// global map does not leak.
func TestHeartbeatTimeoutCleanup(t *testing.T) {
	sess := &mockHeartbeatSession{}
	done := make(chan error, 1)

	err := heartbeat(sess, 10*time.Millisecond, func(err error) { done <- err })
	require.NoError(t, err)
	require.NotZero(t, sess.writtenID, "heartbeat request should have been written")

	select {
	case cbErr := <-done:
		// The timeout branch is what performs the cleanup.
		assert.Equal(t, errHeartbeatReadTimeout, cbErr)
	case <-time.After(2 * time.Second):
		t.Fatal("heartbeat callback was not invoked")
	}

	// After the timeout branch runs (which happens before the callback), the pending
	// response for the heartbeat request must have been removed.
	assert.Nil(t, remoting.GetPendingResponse(remoting.SequenceType(sess.writtenID)),
		"pendingResponses leaked on heartbeat timeout path")
}
