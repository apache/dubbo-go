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

package extension

import (
	"container/list"
	"sync"
	"sync/atomic"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/common"
	commonconfig "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type mockDir struct{}

func (m *mockDir) GetURL() *common.URL                            { return &common.URL{} }
func (m *mockDir) IsAvailable() bool                              { return true }
func (m *mockDir) Destroy()                                       {}
func (m *mockDir) List(invocation base.Invocation) []base.Invoker { return nil }
func (m *mockDir) Subscribe(url *common.URL) error                { return nil }

type mockCustomizer struct{ priority int }

func (m mockCustomizer) GetPriority() int                            { return m.priority }
func (m mockCustomizer) Customize(instance registry.ServiceInstance) {}

type mockServiceNameMapping struct{}

func (m *mockServiceNameMapping) Map(url *common.URL) error { return nil }
func (m *mockServiceNameMapping) Get(url *common.URL, listener mapping.MappingListener) (*gxset.HashSet, error) {
	return nil, nil
}
func (m *mockServiceNameMapping) Remove(url *common.URL) error { return nil }

type mockPostProcessor struct{}

func (m mockPostProcessor) PostProcessReferenceConfig(url *common.URL) {}
func (m mockPostProcessor) PostProcessServiceConfig(url *common.URL)   {}

func TestGetAllCustomShutdownCallbacksReturnsCopy(t *testing.T) {
	customShutdownCallbacksLock.Lock()
	original := customShutdownCallbacks
	customShutdownCallbacks = list.New()
	customShutdownCallbacksLock.Unlock()

	t.Cleanup(func() {
		customShutdownCallbacksLock.Lock()
		customShutdownCallbacks = original
		customShutdownCallbacksLock.Unlock()
	})

	AddCustomShutdownCallback(func() {})
	AddCustomShutdownCallback(func() {})

	callbacks := GetAllCustomShutdownCallbacks()
	assert.Len(t, asSlice(callbacks), 2)

	callbacks.PushBack(func() {})
	callbacksAgain := GetAllCustomShutdownCallbacks()
	assert.Len(t, asSlice(callbacksAgain), 2)
}

func TestGetDirectoryInstanceUsesProtocolAndFallback(t *testing.T) {
	originalDirectories := directories
	directories = NewRegistry[registryDirectory]("registry directory test")
	t.Cleanup(func() {
		directories = originalDirectories
	})

	if oldDefault := defaultDirectory.Load(); oldDefault != nil {
		t.Cleanup(func() { defaultDirectory.Store(oldDefault.(registryDirectory)) })
	} else {
		t.Cleanup(func() { defaultDirectory = atomic.Value{} })
	}

	defaultHit := 0
	protocolHit := 0

	SetDefaultRegistryDirectory(func(url *common.URL, reg registry.Registry) (directory.Directory, error) {
		defaultHit++
		return &mockDir{}, nil
	})

	SetDirectory("polaris", func(url *common.URL, reg registry.Registry) (directory.Directory, error) {
		protocolHit++
		return &mockDir{}, nil
	})

	_, err := GetDirectoryInstance(&common.URL{Protocol: "polaris"}, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, protocolHit)
	assert.Equal(t, 0, defaultHit)

	_, err = GetDirectoryInstance(&common.URL{Protocol: ""}, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, defaultHit)

	_, err = GetDirectoryInstance(&common.URL{Protocol: "unknown"}, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, defaultHit)
}

func TestCustomizersAreSortedAndReturnedAsCopy(t *testing.T) {
	customizersLock.Lock()
	original := customizers
	customizers = make([]registry.ServiceInstanceCustomizer, 0, 8)
	customizersLock.Unlock()

	t.Cleanup(func() {
		customizersLock.Lock()
		customizers = original
		customizersLock.Unlock()
	})

	AddCustomizers(mockCustomizer{priority: 20})
	AddCustomizers(mockCustomizer{priority: 10})

	got := GetCustomizers()
	require.Len(t, got, 2)
	assert.Equal(t, 10, got[0].GetPriority())
	assert.Equal(t, 20, got[1].GetPriority())

	_ = append(got, mockCustomizer{priority: 1})
	assert.Len(t, GetCustomizers(), 2)
}

func TestGlobalServiceNameMappingCreator(t *testing.T) {
	if old := globalNameMappingCreator.Load(); old != nil {
		t.Cleanup(func() { globalNameMappingCreator.Store(old.(ServiceNameMappingCreator)) })
	}

	expected := &mockServiceNameMapping{}
	SetGlobalServiceNameMapping(func() mapping.ServiceNameMapping {
		return expected
	})

	got := GetGlobalServiceNameMapping()
	assert.Same(t, expected, got)
}

func TestConfigPostProcessorRegistrySnapshot(t *testing.T) {
	originalProcessors := processors
	processors = NewRegistry[commonconfig.ConfigPostProcessor]("config post processor test")
	t.Cleanup(func() {
		processors = originalProcessors
	})

	SetConfigPostProcessor("p1", mockPostProcessor{})
	SetConfigPostProcessor("p2", mockPostProcessor{})

	assert.NotNil(t, GetConfigPostProcessor("p1"))
	all := GetConfigPostProcessors()
	assert.Len(t, all, 2)
}

func TestConcurrentCustomShutdownCallbacksAndCustomizers(t *testing.T) {
	customShutdownCallbacksLock.Lock()
	originalCallbacks := customShutdownCallbacks
	customShutdownCallbacks = list.New()
	customShutdownCallbacksLock.Unlock()

	customizersLock.Lock()
	originalCustomizers := customizers
	customizers = make([]registry.ServiceInstanceCustomizer, 0, 8)
	customizersLock.Unlock()

	t.Cleanup(func() {
		customShutdownCallbacksLock.Lock()
		customShutdownCallbacks = originalCallbacks
		customShutdownCallbacksLock.Unlock()

		customizersLock.Lock()
		customizers = originalCustomizers
		customizersLock.Unlock()
	})

	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(p int) {
			defer wg.Done()
			AddCustomShutdownCallback(func() {})
			AddCustomizers(mockCustomizer{priority: p})
			_ = GetAllCustomShutdownCallbacks()
			_ = GetCustomizers()
		}(i)
	}
	wg.Wait()

	assert.Len(t, asSlice(GetAllCustomShutdownCallbacks()), 20)
	assert.Len(t, GetCustomizers(), 20)
}

func asSlice(l *list.List) []any {
	ret := make([]any, 0, l.Len())
	for e := l.Front(); e != nil; e = e.Next() {
		ret = append(ret, e.Value)
	}
	return ret
}
