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
	"fmt"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func TestServiceInstancesChangedListenerAggregatesSameServiceAcrossRevisions(t *testing.T) {
	listener := NewServiceInstancesChangedListener(testApp, gxset.NewSet(testApp))
	notify := &capturingNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)

	instances := []registry.ServiceInstance{
		newTestServiceInstance(t, 20000, "dev"),
		newTestServiceInstance(t, 20001, "pre"),
		newTestServiceInstance(t, 20002, "prod"),
	}
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, instances)))

	assertServiceEvents(t, notify.events, []string{"20000", "20001", "20002"}, []string{"dev", "pre", "prod"})
}

func TestServiceInstancesChangedListenerRefreshesURLsOnProviderRemoveAndRestart(t *testing.T) {
	listener := NewServiceInstancesChangedListener(testApp, gxset.NewSet(testApp))
	notify := &capturingNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)

	dev := newTestServiceInstance(t, 20000, "dev")
	pre := newTestServiceInstance(t, 20001, "pre")
	prod := newTestServiceInstance(t, 20002, "prod")

	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		dev,
		pre,
		prod,
	})))
	assertServiceEvents(t, notify.events, []string{"20000", "20001", "20002"}, []string{"dev", "pre", "prod"})

	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		dev,
		prod,
	})))
	assertServiceEvents(t, notify.events, []string{"20000", "20002"}, []string{"dev", "prod"})

	restartedPre := newTestServiceInstanceWithRevision(t, 20001, "pre-restarted", "rev-20001-restarted")
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		dev,
		restartedPre,
		prod,
	})))
	assertServiceEvents(t, notify.events, []string{"20000", "20001", "20002"}, []string{"dev", "pre-restarted", "prod"})
}

func TestServiceInstancesChangedListenerRefreshesAndClearsEnvironmentWhenRevisionIsUnchanged(t *testing.T) {
	listener := NewServiceInstancesChangedListener(testApp, gxset.NewSet(testApp))
	notify := &capturingNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)

	revision := "rev-20001-same-environment-change"
	metaCache.Set(revision, newTestMetadataInfo(t, revision, 20001, "pre"))

	pre := newTestServiceInstanceOnly(20001, "pre", revision)
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		pre,
	})))
	assertServiceEvents(t, notify.events, []string{"20001"}, []string{"pre"})

	restartedPre := newTestServiceInstanceOnly(20001, "pre-restarted", revision)
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		restartedPre,
	})))
	assertServiceEvents(t, notify.events, []string{"20001"}, []string{"pre-restarted"})

	clearedPre := newTestServiceInstanceOnly(20001, "", revision)
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		clearedPre,
	})))
	assertServiceEvents(t, notify.events, []string{"20001"}, []string{""})
	assertServiceEventEnvironmentsAbsent(t, notify.events)

	removedPre := newTestServiceInstanceOnly(20001, "ignored", revision)
	delete(removedPre.GetMetadata(), constant.EnvironmentKey)
	require.NoError(t, listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
		removedPre,
	})))
	assertServiceEvents(t, notify.events, []string{"20001"}, []string{""})
	assertServiceEventEnvironmentsAbsent(t, notify.events)
}

func TestServiceInstancesChangedListenerSkipsNilMetadataWithoutPanic(t *testing.T) {
	listener := NewServiceInstancesChangedListener(testApp, gxset.NewSet(testApp))
	notify := &capturingNotifyListener{}
	listener.AddListenerAndNotify(common.MatchKey(testInterface, constant.TriProtocol), notify)

	revision := "rev-20003-nil-metadata"
	var metadataInfo *info.MetadataInfo
	metaCache.Set(revision, metadataInfo)
	t.Cleanup(func() {
		metaCache.Delete(revision)
	})

	instance := newTestServiceInstanceOnly(20003, "pre", revision)
	var err error
	require.NotPanics(t, func() {
		err = listener.OnEvent(registry.NewServiceInstancesChangedEvent(testApp, []registry.ServiceInstance{
			instance,
		}))
	})
	require.NoError(t, err)
	assert.Empty(t, notify.events)
}

func TestCreateInstanceCarriesEnvironmentMetadata(t *testing.T) {
	meta := info.NewMetadataInfo(testApp, "")
	providerURL := newTestProviderURL(t, 20001, "pre")

	instance := createInstance(meta, providerURL)

	assert.Equal(t, "pre", instance.GetMetadata()[constant.EnvironmentKey])
}

func newTestServiceInstance(t *testing.T, port int, environment string) registry.ServiceInstance {
	t.Helper()

	return newTestServiceInstanceWithRevision(t, port, environment, fmt.Sprintf("rev-%d", port))
}

func newTestServiceInstanceWithRevision(t *testing.T, port int, environment string, revision string) registry.ServiceInstance {
	t.Helper()

	metaCache.Set(revision, newTestMetadataInfo(t, revision, port, environment))
	return newTestServiceInstanceOnly(port, environment, revision)
}

func newTestServiceInstanceOnly(port int, environment string, revision string) registry.ServiceInstance {
	return &registry.DefaultServiceInstance{
		ID:          fmt.Sprintf("127.0.0.1:%d", port),
		ServiceName: testApp,
		Host:        "127.0.0.1",
		Port:        port,
		Enable:      true,
		Healthy:     true,
		Metadata: map[string]string{
			constant.ExportedServicesRevisionPropertyName: revision,
			constant.ServiceInstanceEndpoints:             fmt.Sprintf(`[{"port":%d,"protocol":"tri"}]`, port),
			constant.EnvironmentKey:                       environment,
		},
	}
}

func newTestMetadataInfo(t *testing.T, revision string, port int, environment string) *info.MetadataInfo {
	t.Helper()

	serviceURL := newTestProviderURL(t, port, environment)
	service := info.NewServiceInfoWithURL(serviceURL)
	return info.NewMetadataInfoWithParams(testApp, revision, map[string]*info.ServiceInfo{
		service.GetMatchKey(): service,
	})
}

func newTestProviderURL(t *testing.T, port int, environment string) *common.URL {
	t.Helper()

	serviceURL, err := common.NewURL(
		fmt.Sprintf("tri://127.0.0.1:%d/%s", port, testInterface),
		common.WithInterface(testInterface),
		common.WithMethods([]string{"Greet"}),
		common.WithParamsValue(constant.ApplicationKey, testApp),
		common.WithParamsValue(constant.EnvironmentKey, environment),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)
	require.NoError(t, err)
	return serviceURL
}

func assertServiceEvents(t *testing.T, events []*registry.ServiceEvent, ports, environments []string) {
	t.Helper()

	require.Len(t, events, len(ports))
	require.Len(t, environments, len(ports))
	assert.ElementsMatch(t, expectedServiceEvents(ports, environments), actualServiceEvents(events))
}

func assertServiceEventEnvironmentsAbsent(t *testing.T, events []*registry.ServiceEvent) {
	t.Helper()

	for _, event := range events {
		assert.NotContains(t, event.Service.GetParams(), constant.EnvironmentKey)
	}
}

type serviceEventAssertion struct {
	port        string
	environment string
}

func expectedServiceEvents(ports, environments []string) []serviceEventAssertion {
	events := make([]serviceEventAssertion, 0, len(ports))
	for i, port := range ports {
		events = append(events, serviceEventAssertion{
			port:        port,
			environment: environments[i],
		})
	}
	return events
}

func actualServiceEvents(events []*registry.ServiceEvent) []serviceEventAssertion {
	actual := make([]serviceEventAssertion, 0, len(events))
	for _, event := range events {
		actual = append(actual, serviceEventAssertion{
			port:        event.Service.Port,
			environment: event.Service.GetParam(constant.EnvironmentKey, ""),
		})
	}
	return actual
}

type capturingNotifyListener struct {
	events []*registry.ServiceEvent
}

func (c *capturingNotifyListener) Notify(event *registry.ServiceEvent) {
	c.events = append(c.events, event)
}

func (c *capturingNotifyListener) NotifyAll(events []*registry.ServiceEvent, callback func()) {
	c.events = append([]*registry.ServiceEvent(nil), events...)
	if callback != nil {
		callback()
	}
}
