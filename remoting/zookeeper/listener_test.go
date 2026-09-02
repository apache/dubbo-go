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

package zookeeper

import (
	"net/url"
	"testing"
	"time"
)

import (
	gxzookeeper "github.com/dubbogo/gost/database/kv/zk"

	"github.com/go-zookeeper/zk"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type recordingDataListener struct {
	events []remoting.Event
}

func (l *recordingDataListener) DataChange(event remoting.Event) bool {
	l.events = append(l.events, event)
	return true
}

type recordingWatchStateListener struct {
	registerWatch bool
	result        WatchRegistrationResult
	changed       int
	registered    int
	failed        int
}

func (l *recordingWatchStateListener) WatchStateChanged(string) bool {
	l.changed++
	return l.registerWatch
}

func (l *recordingWatchStateListener) WatchRegistered(string, <-chan zk.Event, int64, int64) WatchRegistrationResult {
	l.registered++
	return l.result
}

func (l *recordingWatchStateListener) WatchStateChangeFailed(string) {
	l.failed++
}

func TestZkPath(t *testing.T) {
	zkPath := "io.grpc.examples.helloworld.GreeterGrpc$IGreeter"
	zkPath = url.QueryEscape(zkPath)
	assert.Equal(t, "io.grpc.examples.helloworld.GreeterGrpc%24IGreeter", zkPath)
}

func TestListenConfigurationEventStopsOnClose(t *testing.T) {
	client, err := gxzookeeper.NewZookeeperClient(
		"remoting-listener-event-loop",
		[]string{"127.0.0.1:0"},
		false,
		gxzookeeper.WithZkTimeOut(100*time.Millisecond),
	)
	require.NoError(t, err)
	defer client.Close()

	eventListener := NewZkEventListener(client)
	eventListener.ListenConfigurationEvent("/config", &recordingDataListener{})
	// ListenConfigurationEvent registers asynchronously before waiting on exit.
	time.Sleep(50 * time.Millisecond)

	closed := make(chan struct{})
	go func() {
		eventListener.Close()
		close(closed)
	}()
	select {
	case <-closed:
	case <-time.After(2 * time.Second):
		t.Fatal("ListenConfigurationEvent goroutine did not exit after Close")
	}
}

func TestProcessConfigurationEventRejectsUnstableMissingNodeRead(t *testing.T) {
	dataListener := &recordingDataListener{}
	watchListener := &recordingWatchStateListener{}
	reader := configurationEventReader{
		exists: func(bool) (bool, <-chan zk.Event, int64, int64, error) {
			return false, nil, 1, 2, nil
		},
		content: func() ([]byte, int64, int64, error) {
			t.Fatalf("content read should not happen for a missing node")
			return nil, 0, 0, nil
		},
	}

	processConfigurationEvent(
		zk.Event{Type: zk.EventNodeDataChanged, Path: "/config"},
		dataListener,
		watchListener,
		true,
		reader,
	)

	require.Empty(t, dataListener.events)
	require.Equal(t, 1, watchListener.failed)
}

func TestProcessConfigurationEventHandlesMissingNodeAndWatchResults(t *testing.T) {
	t.Run("ttl fallback reports stable deletion", func(t *testing.T) {
		dataListener := &recordingDataListener{}
		watchListener := &recordingWatchStateListener{}
		processConfigurationEvent(
			zk.Event{Type: zk.EventNodeDataChanged, Path: "/config"},
			dataListener,
			watchListener,
			true,
			configurationEventReader{
				exists: func(register bool) (bool, <-chan zk.Event, int64, int64, error) {
					require.False(t, register)
					return false, nil, 1, 1, nil
				},
				content: func() ([]byte, int64, int64, error) {
					t.Fatalf("content read should not happen for a missing node")
					return nil, 0, 0, nil
				},
			},
		)

		require.Len(t, dataListener.events, 1)
		require.Equal(t, remoting.EventTypeDel, dataListener.events[0].Action)
		require.Zero(t, watchListener.failed)
	})

	t.Run("accepted watch reports updated content", func(t *testing.T) {
		dataListener := &recordingDataListener{}
		watchListener := &recordingWatchStateListener{
			registerWatch: true,
			result:        WatchRegistrationAccepted,
		}
		processConfigurationEvent(
			zk.Event{Type: zk.EventNodeDataChanged, Path: "/config"},
			dataListener,
			watchListener,
			true,
			configurationEventReader{
				exists: func(register bool) (bool, <-chan zk.Event, int64, int64, error) {
					require.True(t, register)
					return true, make(chan zk.Event), 1, 1, nil
				},
				content: func() ([]byte, int64, int64, error) {
					return []byte("value"), 1, 1, nil
				},
			},
		)

		require.Len(t, dataListener.events, 1)
		require.Equal(t, remoting.EventTypeAdd, dataListener.events[0].Action)
		require.Equal(t, "value", dataListener.events[0].Content)
		require.Equal(t, 1, watchListener.registered)
	})

	t.Run("discarded watch does not notify", func(t *testing.T) {
		dataListener := &recordingDataListener{}
		watchListener := &recordingWatchStateListener{
			registerWatch: true,
			result:        WatchRegistrationDiscarded,
		}
		processConfigurationEvent(
			zk.Event{Type: zk.EventNodeDataChanged, Path: "/config"},
			dataListener,
			watchListener,
			true,
			configurationEventReader{
				exists: func(bool) (bool, <-chan zk.Event, int64, int64, error) {
					return true, make(chan zk.Event), 1, 1, nil
				},
				content: func() ([]byte, int64, int64, error) {
					t.Fatalf("content read should not happen for a discarded watch")
					return nil, 0, 0, nil
				},
			},
		)

		require.Empty(t, dataListener.events)
	})

	t.Run("reload uses an ordinary stable existence read", func(t *testing.T) {
		dataListener := &recordingDataListener{}
		watchListener := &recordingWatchStateListener{
			registerWatch: true,
			result:        WatchRegistrationReload,
		}
		var calls int
		processConfigurationEvent(
			zk.Event{Type: zk.EventNodeDeleted, Path: "/config"},
			dataListener,
			watchListener,
			true,
			configurationEventReader{
				exists: func(register bool) (bool, <-chan zk.Event, int64, int64, error) {
					calls++
					if calls == 1 {
						require.True(t, register)
						return true, make(chan zk.Event), 1, 1, nil
					}
					require.False(t, register)
					return false, nil, 1, 1, nil
				},
				content: func() ([]byte, int64, int64, error) {
					t.Fatalf("content read should not happen after reload reports missing")
					return nil, 0, 0, nil
				},
			},
		)

		require.Len(t, dataListener.events, 1)
		require.Equal(t, remoting.EventTypeDel, dataListener.events[0].Action)
		require.Equal(t, 2, calls)
	})
}
