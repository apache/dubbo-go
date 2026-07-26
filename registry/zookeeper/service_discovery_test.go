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
	"reflect"
	"sync"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func Test_newZookeeperServiceDiscovery(t *testing.T) {
	url, _ := common.NewURL("dubbo://127.0.0.1:2181",
		common.WithParamsValue(constant.ClientNameKey, "zk-client"))
	sd, err := newZookeeperServiceDiscovery(url)
	require.NoError(t, err)
	err = sd.Destroy()
	require.NoError(t, err)

}
func Test_zookeeperServiceDiscovery_DataChange(t *testing.T) {
	serviceDiscovery := &zookeeperServiceDiscovery{}
	assert.Equal(t, registry.DefaultPageSize, serviceDiscovery.GetDefaultPageSize())
}

// mockChangedListener is a minimal ServiceInstancesChangedListener for tests.
type mockChangedListener struct{ name string }

func (m *mockChangedListener) OnEvent(observer.Event) error                         { return nil }
func (m *mockChangedListener) AddListenerAndNotify(string, registry.NotifyListener) {}
func (m *mockChangedListener) RemoveListener(string)                                {}
func (m *mockChangedListener) GetServiceNames() *gxset.HashSet                      { return gxset.NewSet(m.name) }
func (m *mockChangedListener) Accept(observer.Event) bool                           { return true }
func (m *mockChangedListener) GetEventType() reflect.Type                           { return nil }
func (m *mockChangedListener) GetPriority() int                                     { return 0 }

// TestZookeeperServiceDiscovery_SnapshotListeners verifies the listener map is
// read under listenLock and a missing key is a safe no-op. Regression for #3512
// (unlocked map read racing AddListener/Destroy).
func TestZookeeperServiceDiscovery_SnapshotListeners(t *testing.T) {
	sd := &zookeeperServiceDiscovery{
		instanceListenerMap: map[string]*gxset.HashSet{
			"svc-a": gxset.NewSet(&mockChangedListener{name: "a1"}, &mockChangedListener{name: "a2"}),
		},
	}
	assert.Nil(t, sd.snapshotListeners("missing"))
	assert.Len(t, sd.snapshotListeners("svc-a"), 2)

	// Concurrent snapshot vs concurrent map mutation under listenLock must be race-free.
	var wg sync.WaitGroup
	for range 50 {
		wg.Add(2)
		go func() { defer wg.Done(); _ = sd.snapshotListeners("svc-a") }()
		go func() {
			defer wg.Done()
			sd.listenLock.Lock()
			sd.instanceListenerMap["svc-a"] = gxset.NewSet(&mockChangedListener{name: "x"})
			sd.listenLock.Unlock()
		}()
	}
	wg.Wait()
}
