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
	"fmt"
	"testing"
	"time"
)

import (
	"github.com/samuel/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
)

func verifyEventStateOrder(t *testing.T, c <-chan zk.Event, expectedStates []zk.State, source string) {
	for _, state := range expectedStates {
		for {
			event, ok := <-c
			if !ok {
				t.Fatalf("unexpected channel close for %s", source)
			}
			fmt.Println(event)
			if event.Type != zk.EventSession {
				continue
			}

			if event.State != state {
				t.Fatalf("mismatched state order from %s, expected %v, received %v", source, state, event.State)
			}
			break
		}
	}
}

//func Test_newZookeeperClient(t *testing.T) {
//	ts, err := zk.StartTestCluster(1, nil, nil)
//	if err != nil {
//		t.Fatal(err)
//	}
//	defer ts.Stop()
//
//	callbackChan := make(chan zk.Event)
//	f := func(event zk.Event) {
//		callbackChan <- event
//	}
//
//	zook, eventChan, err := ts.ConnectWithOptions(15*time.Second, zk.WithEventCallback(f))
//	if err != nil {
//		t.Fatalf("Connect returned error: %+v", err)
//	}
//
//	states := []zk.State{zk.StateConnecting, zk.StateConnected, zk.StateHasSession}
//	verifyEventStateOrder(t, callbackChan, states, "callback")
//	verifyEventStateOrder(t, eventChan, states, "event channel")
//
//	zook.Close()
//	verifyEventStateOrder(t, callbackChan, []zk.State{zk.StateDisconnected}, "callback")
//	verifyEventStateOrder(t, eventChan, []zk.State{zk.StateDisconnected}, "event channel")
//
//}

func Test_newMockZookeeperClient(t *testing.T) {
	ts, z, event, err := NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)
	defer ts.Stop()
	states := []zk.State{zk.StateConnecting, zk.StateConnected, zk.StateHasSession}
	verifyEventStateOrder(t, event, states, "event channel")

	z.Close()
	verifyEventStateOrder(t, event, []zk.State{zk.StateDisconnected}, "event channel")
}

func TestCreate(t *testing.T) {
	ts, z, event, _ := NewMockZookeeperClient("test", 15*time.Second)
	defer ts.Stop()
	err := z.Create("test1/test2/test3/test4")
	assert.NoError(t, err)

	states := []zk.State{zk.StateConnecting, zk.StateConnected, zk.StateHasSession}
	verifyEventStateOrder(t, event, states, "event channel")
}

func TestCreateDelete(t *testing.T) {
	ts, z, event, _ := NewMockZookeeperClient("test", 15*time.Second)
	defer ts.Stop()

	states := []zk.State{zk.StateConnecting, zk.StateConnected, zk.StateHasSession}
	verifyEventStateOrder(t, event, states, "event channel")
	err := z.Create("/test1/test2/test3/test4")
	assert.NoError(t, err)
	err2 := z.Delete("/test1/test2/test3/test4")
	assert.NoError(t, err2)
	//verifyEventOrder(t, event, []zk.EventType{zk.EventNodeCreated}, "event channel")
}

func TestRegisterTemp(t *testing.T) {
	ts, z, event, _ := NewMockZookeeperClient("test", 15*time.Second)
	defer ts.Stop()
	err := z.Create("/test1/test2/test3")
	assert.NoError(t, err)

	tmpath, err := z.RegisterTemp("/test1/test2/test3", "test4")
	assert.NoError(t, err)
	assert.Equal(t, "/test1/test2/test3/test4", tmpath)
	states := []zk.State{zk.StateConnecting, zk.StateConnected, zk.StateHasSession}
	verifyEventStateOrder(t, event, states, "event channel")
}

func TestRegisterTempSeq(t *testing.T) {
	ts, z, event, _ := NewMockZookeeperClient("test", 15*time.Second)
	defer ts.Stop()
	err := z.Create("/test1/test2/test3")
	assert.NoError(t, err)
	tmpath, err := z.RegisterTempSeq("/test1/test2/test3", []byte("test"))
	assert.NoError(t, err)
	assert.Equal(t, "/test1/test2/test3/0000000000", tmpath)
	states := []zk.State{zk.StateConnecting, zk.StateConnected, zk.StateHasSession}
	verifyEventStateOrder(t, event, states, "event channel")
}
