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

package servicediscovery

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	perrors "github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// destroyRaceDiscovery is a concurrency-safe ServiceDiscovery stub that counts
// the calls a late subscribe must not make after Destroy.
type destroyRaceDiscovery struct {
	mockServiceDiscovery
	mu                sync.Mutex
	instances         []registry.ServiceInstance
	getInstancesCalls int
	addListenerCalls  int
}

func (m *destroyRaceDiscovery) GetInstances(string) []registry.ServiceInstance {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getInstancesCalls++
	return append([]registry.ServiceInstance(nil), m.instances...)
}

func (m *destroyRaceDiscovery) AddListener(registry.ServiceInstancesChangedListener) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.addListenerCalls++
	return nil
}

func (m *destroyRaceDiscovery) counts() (getInstances, addListener int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getInstancesCalls, m.addListenerCalls
}

func newDestroyRaceRegistry(sd registry.ServiceDiscovery) *serviceDiscoveryRegistry {
	registryURL, _ := common.NewURL(testRegistryURL,
		common.WithParamsValue(constant.RegistryKey, "mock"))
	return &serviceDiscoveryRegistry{
		url:              registryURL,
		serviceDiscovery: sd,
		serviceListeners: make(map[string]registry.ServiceInstancesChangedListener),
	}
}

// TestSubscribeURLDiscardedWhenDestroyRacesInitialLoad is the deterministic
// interleaving test for the review finding: Destroy runs while SubscribeURL is
// still in its initial GetInstances/metadata phase (invisible to Destroy's
// listener sweep). The late subscribe must be discarded at install time: no
// listener installed, no AddListener, and no retry timer probing metadata
// after Destroy.
func TestSubscribeURLDiscardedWhenDestroyRacesInitialLoad(t *testing.T) {
	const revision = "rev-destroy-race"
	const port = 22401
	stubRetryDelays(t, 5*time.Millisecond, 20*time.Millisecond)

	entered := make(chan struct{})
	release := make(chan struct{})
	var enteredOnce sync.Once
	var fetchCalls atomic.Int32
	stubMetadataFetch(t, func(context.Context, string, registry.ServiceInstance, string, string) (*info.MetadataInfo, error) {
		fetchCalls.Add(1)
		enteredOnce.Do(func() { close(entered) })
		<-release
		return nil, perrors.New("metadata unreachable")
	})

	sd := &destroyRaceDiscovery{
		instances: []registry.ServiceInstance{newTestServiceInstanceOnly(port, "", revision)},
	}
	reg := newDestroyRaceRegistry(sd)

	consumerURL, err := common.NewURL("tri://127.0.0.1:20000/",
		common.WithInterface(testInterface),
		common.WithParamsValue(constant.SideKey, constant.SideConsumer))
	require.NoError(t, err)

	subscribeDone := make(chan struct{})
	go func() {
		defer close(subscribeDone)
		reg.SubscribeURL(consumerURL, &retryNotifyListener{}, gxset.NewSet(testApp))
	}()

	// SubscribeURL is now blocked inside the metadata fetch of its initial
	// load phase, invisible to Destroy's listener sweep.
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("SubscribeURL never reached the metadata fetch")
	}
	reg.Destroy()
	close(release)
	select {
	case <-subscribeDone:
	case <-time.After(2 * time.Second):
		t.Fatal("SubscribeURL did not return after the blocked fetch was released")
	}

	assert.Nil(t, reg.getServiceListener(testApp), "late listener must not be installed after Destroy")
	_, addListenerCalls := sd.counts()
	assert.Equal(t, 0, addListenerCalls, "AddListener must not be called after Destroy")

	// The in-flight fetch may complete, but no retry timer may probe further.
	before := fetchCalls.Load()
	time.Sleep(150 * time.Millisecond)
	assert.Equal(t, before, fetchCalls.Load(), "no metadata fetch may happen after Destroy")
}

// TestSubscribeURLAfterDestroyIsNoop verifies a subscribe issued after Destroy
// does not touch the service discovery at all.
func TestSubscribeURLAfterDestroyIsNoop(t *testing.T) {
	sd := &destroyRaceDiscovery{}
	reg := newDestroyRaceRegistry(sd)
	reg.Destroy()

	consumerURL, err := common.NewURL("tri://127.0.0.1:20000/",
		common.WithInterface(testInterface),
		common.WithParamsValue(constant.SideKey, constant.SideConsumer))
	require.NoError(t, err)
	reg.SubscribeURL(consumerURL, &retryNotifyListener{}, gxset.NewSet(testApp))

	getInstancesCalls, addListenerCalls := sd.counts()
	assert.Equal(t, 0, getInstancesCalls, "destroyed registry must not query instances")
	assert.Equal(t, 0, addListenerCalls, "destroyed registry must not add listeners")
	assert.Nil(t, reg.getServiceListener(testApp), "destroyed registry must not install listeners")
}
