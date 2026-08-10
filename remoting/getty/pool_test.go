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
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	gettylib "github.com/apache/dubbo-getty"

	perrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

func TestGettyRPCClientUpdateActive(t *testing.T) {
	client := &gettyRPCClient{}
	client.updateActive(1234567890)
	assert.Equal(t, int64(1234567890), client.active.Load())

	client.updateActive(0)
	assert.Equal(t, int64(0), client.active.Load())
}

func TestGettyRPCClientSelectSession(t *testing.T) {
	client := &gettyRPCClient{sessions: nil}
	assert.Nil(t, client.selectSession())

	client.sessions = []*rpcSession{}
	assert.Nil(t, client.selectSession())
}

func TestGettyRPCClientSessionOperations(t *testing.T) {
	client := &gettyRPCClient{}

	// Remove/update nil session should not panic
	client.removeSession(nil)
	client.updateSession(nil)

	// Get from nil sessions
	_, err := client.getClientRpcSession(nil)
	assert.Equal(t, errClientClosed, err)

	// Session not found
	client.sessions = []*rpcSession{}
	_, err = client.getClientRpcSession(nil)
	assert.Contains(t, err.Error(), "session not exist")
}

func TestGettyRPCClientIsAvailable(t *testing.T) {
	client := &gettyRPCClient{sessions: nil}
	assert.False(t, client.isAvailable())

	client.sessions = []*rpcSession{}
	assert.False(t, client.isAvailable())
}

func TestGettyRPCClientClose(t *testing.T) {
	client := &gettyRPCClient{sessions: []*rpcSession{}}
	require.NoError(t, client.close())
	err := client.close()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "close gettyRPCClient")
	assert.Contains(t, err.Error(), "again")
}

func TestGettyRPCClientConcurrent(t *testing.T) {
	client := &gettyRPCClient{sessions: []*rpcSession{}}
	var wg sync.WaitGroup

	for i := range 50 {
		wg.Add(1)
		go func(val int64) {
			defer wg.Done()
			client.updateActive(val)
			_ = client.selectSession()
			_ = client.isAvailable()
		}(int64(i))
	}
	wg.Wait()
}

func TestRpcSession(t *testing.T) {
	s := &rpcSession{reqNum: 0}

	s.AddReqNum(5)
	assert.Equal(t, int32(5), s.GetReqNum())

	s.AddReqNum(3)
	assert.Equal(t, int32(8), s.GetReqNum())

	s.AddReqNum(-3)
	assert.Equal(t, int32(5), s.GetReqNum())
}

func TestRpcSessionConcurrent(t *testing.T) {
	s := &rpcSession{}
	var wg sync.WaitGroup

	for range 100 {
		wg.Go(func() {
			s.AddReqNum(1)
		})
	}
	wg.Wait()
	assert.Equal(t, int32(100), s.GetReqNum())
}

func TestGettyRPCClientLifecycle(t *testing.T) {
	client := &gettyRPCClient{addr: "127.0.0.1:20880", sessions: []*rpcSession{}}

	assert.False(t, client.isAvailable())
	assert.Equal(t, int64(0), client.active.Load())

	client.updateActive(1234567890)
	assert.Equal(t, int64(1234567890), client.active.Load())

	require.NoError(t, client.close())
	assert.Equal(t, int64(0), client.active.Load())
}

type stubSession struct {
	closed    atomic.Bool
	writes    atomic.Int32
	onClose   func()
	closeOnce sync.Once
}

func (s *stubSession) ID() uint32                              { return 1 }
func (s *stubSession) SetCompressType(gettylib.CompressType)   {}
func (s *stubSession) LocalAddr() string                       { return "127.0.0.1:12345" }
func (s *stubSession) RemoteAddr() string                      { return "127.0.0.1:20880" }
func (s *stubSession) IncReadPkgNum()                          {}
func (s *stubSession) IncWritePkgNum()                         {}
func (s *stubSession) UpdateActive()                           {}
func (s *stubSession) GetActive() time.Time                    { return time.Now() }
func (s *stubSession) ReadTimeout() time.Duration              { return time.Second }
func (s *stubSession) SetReadTimeout(time.Duration)            {}
func (s *stubSession) WriteTimeout() time.Duration             { return time.Second }
func (s *stubSession) SetWriteTimeout(time.Duration)           {}
func (s *stubSession) Send(any) (int, error)                   { return 0, nil }
func (s *stubSession) CloseConn(int)                           {}
func (s *stubSession) SetSession(gettylib.Session)             {}
func (s *stubSession) Reset()                                  {}
func (s *stubSession) Conn() net.Conn                          { return nil }
func (s *stubSession) Stat() string                            { return "stub-session" }
func (s *stubSession) IsClosed() bool                          { return s.closed.Load() }
func (s *stubSession) EndPoint() gettylib.EndPoint             { return nil }
func (s *stubSession) SetMaxMsgLen(int)                        {}
func (s *stubSession) SetName(string)                          {}
func (s *stubSession) SetEventListener(gettylib.EventListener) {}
func (s *stubSession) SetPkgHandler(gettylib.ReadWriter)       {}
func (s *stubSession) SetReader(gettylib.Reader)               {}
func (s *stubSession) SetWriter(gettylib.Writer)               {}
func (s *stubSession) SetCronPeriod(int)                       {}
func (s *stubSession) SetWaitTime(time.Duration)               {}
func (s *stubSession) GetAttribute(any) any                    { return nil }
func (s *stubSession) SetAttribute(any, any)                   {}
func (s *stubSession) RemoveAttribute(any)                     {}
func (s *stubSession) WritePkg(pkg any, timeout time.Duration) (int, int, error) {
	s.writes.Add(1)
	return 1, 1, nil
}
func (s *stubSession) WriteBytes([]byte) (int, error)         { return 0, nil }
func (s *stubSession) WriteBytesArray(...[]byte) (int, error) { return 0, nil }
func (s *stubSession) Close() {
	s.closeOnce.Do(func() {
		s.closed.Store(true)
		if s.onClose != nil {
			s.onClose()
		}
	})
}

// This test case verifies the scenario described at https://github.com/apache/dubbo-go/issues/3509.
func TestReadTimeoutRemovesHalfDeadSession(t *testing.T) {
	sess := &stubSession{}
	client := &Client{addr: "127.0.0.1:20880"}
	rpcClient := &gettyRPCClient{rpcClient: client, sessions: []*rpcSession{{session: sess}}}
	client.gettyClient = rpcClient
	client.gettyClientCreated.Store(true)

	req := remoting.NewRequest("2.0.2")
	req.TwoWay = true
	rsp := remoting.NewPendingResponse(req.ID)
	remoting.AddPendingResponse(rsp)

	err := client.Request(req, 10*time.Millisecond, rsp)
	require.Error(t, err)
	require.ErrorIs(t, err, errClientReadTimeout)
	assert.Eventually(t, sess.IsClosed, time.Second, time.Millisecond, "timed out session should be closed")
	assert.Equal(t, int32(1), sess.writes.Load())

	assert.Nil(t, rpcClient.selectSession())
	assert.Empty(t, rpcClient.sessions)
	assert.Nil(t, client.gettyClient, "the connection handle should be reset after the last session is removed")
	assert.Nil(t, remoting.GetPendingResponse(remoting.SequenceType(req.ID)))
	assert.False(t, client.clientClosed, "a timed out session must not close the client")
}

func TestIssueClosedSessionIsNotSelected(t *testing.T) {
	sess := &stubSession{}
	sess.Close()
	client := &gettyRPCClient{sessions: []*rpcSession{{session: sess}}}

	selected := client.selectSession()
	assert.Nil(t, selected)
}

func TestDelayedOnCloseDoesNotResetReplacement(t *testing.T) {
	client := &Client{addr: "127.0.0.1:20880"}
	oldSession := &stubSession{}
	oldPool := &gettyRPCClient{rpcClient: client, sessions: []*rpcSession{{session: oldSession}}}
	client.gettyClient = oldPool
	client.gettyClientCreated.Store(true)

	closeStarted := make(chan struct{})
	allowOnClose := make(chan struct{})
	onCloseDone := make(chan struct{})
	oldSession.onClose = func() {
		close(closeStarted)
		<-allowOnClose
		oldPool.removeSession(oldSession)
		close(onCloseDone)
	}

	request := remoting.NewRequest("2.0.2")
	request.TwoWay = true
	response := remoting.NewPendingResponse(request.ID)
	remoting.AddPendingResponse(response)

	err := client.Request(request, 10*time.Millisecond, response)
	require.ErrorIs(t, err, errClientReadTimeout)
	<-closeStarted

	replacement := &gettyRPCClient{rpcClient: client, sessions: []*rpcSession{{session: &stubSession{}}}}
	client.gettyClientMux.Lock()
	client.gettyClient = replacement
	client.gettyClientCreated.Store(true)
	client.gettyClientMux.Unlock()

	client.resetRpcConn(oldPool)
	client.gettyClientMux.RLock()
	assert.Same(t, replacement, client.gettyClient)
	assert.True(t, client.gettyClientCreated.Load())
	client.gettyClientMux.RUnlock()

	close(allowOnClose)
	select {
	case <-onCloseDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delayed OnClose")
	}

	client.gettyClientMux.RLock()
	assert.Same(t, replacement, client.gettyClient)
	assert.True(t, client.gettyClientCreated.Load())
	client.gettyClientMux.RUnlock()
}

func TestTimeoutResetWithConcurrentSelectSession(t *testing.T) {
	client := &Client{addr: "127.0.0.1:20880"}
	oldSession := &stubSession{}
	oldPool := &gettyRPCClient{rpcClient: client, sessions: []*rpcSession{{session: oldSession}}}
	client.gettyClient = oldPool
	client.gettyClientCreated.Store(true)

	closeStarted := make(chan struct{})
	allowOnClose := make(chan struct{})
	onCloseDone := make(chan struct{})
	oldSession.onClose = func() {
		close(closeStarted)
		<-allowOnClose
		oldPool.removeSession(oldSession)
		close(onCloseDone)
	}

	request := remoting.NewRequest("2.0.2")
	request.TwoWay = true
	response := remoting.NewPendingResponse(request.ID)
	remoting.AddPendingResponse(response)
	require.ErrorIs(t, client.Request(request, 10*time.Millisecond, response), errClientReadTimeout)
	<-closeStarted
	client.gettyClientMux.RLock()
	assert.Nil(t, client.gettyClient)
	assert.False(t, client.gettyClientCreated.Load())
	client.gettyClientMux.RUnlock()

	replacementSession := &stubSession{}
	replacement := &gettyRPCClient{rpcClient: client, sessions: []*rpcSession{{session: replacementSession}}}
	client.gettyClientMux.Lock()
	client.gettyClient = replacement
	client.gettyClientCreated.Store(true)
	client.gettyClientMux.Unlock()

	selectDone := make(chan struct{})
	selectErr := make(chan error, 1)
	go func() {
		defer close(selectDone)
		for range 100 {
			selectedClient, selectedSession, err := client.selectSession(client.addr)
			if err != nil {
				selectErr <- err
				return
			}
			if selectedClient != replacement || selectedSession != replacementSession {
				selectErr <- perrors.New("selectSession returned a stale connection")
				return
			}
		}
	}()

	close(allowOnClose)
	select {
	case err := <-selectErr:
		t.Fatal(err)
	case <-selectDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for concurrent selectSession")
	}
	select {
	case <-onCloseDone:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for delayed OnClose")
	}
}
