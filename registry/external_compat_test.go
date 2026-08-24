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

package registry_test

import (
	"testing"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/hash/page"

	"github.com/stretchr/testify/assert"

	uberatomic "go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// externalServiceDiscovery mimics a third-party ServiceDiscovery implementation
// written against the released v3 API: its pagination methods still return
// gxpage.Pager instead of the local registry.Pager. The compile-time assertion
// below guards the exported interface signature across the v3.x line.
type externalServiceDiscovery struct{}

var _ registry.ServiceDiscovery = (*externalServiceDiscovery)(nil)

func (externalServiceDiscovery) String() string { return "external" }

func (externalServiceDiscovery) Destroy() error { return nil }

func (externalServiceDiscovery) Register(instance registry.ServiceInstance) error { return nil }

func (externalServiceDiscovery) Update(instance registry.ServiceInstance) error { return nil }

func (externalServiceDiscovery) Unregister(instance registry.ServiceInstance) error { return nil }

func (externalServiceDiscovery) GetDefaultPageSize() int { return 100 }

func (externalServiceDiscovery) GetServices() *gxset.HashSet { return nil }

func (externalServiceDiscovery) GetInstances(serviceName string) []registry.ServiceInstance {
	return nil
}

func (externalServiceDiscovery) GetInstancesByPage(serviceName string, offset int, pageSize int) gxpage.Pager {
	return nil
}

func (externalServiceDiscovery) GetHealthyInstancesByPage(serviceName string, offset int, pageSize int, healthy bool) gxpage.Pager {
	return nil
}

func (externalServiceDiscovery) GetRequestInstances(serviceNames []string, offset int, requestedSize int) map[string]gxpage.Pager {
	return nil
}

func (externalServiceDiscovery) AddListener(listener registry.ServiceInstancesChangedListener) error {
	return nil
}

func TestExternalServiceDiscoveryStillSatisfiesInterface(t *testing.T) {
	// The compile-time assertion above guarantees the released implementation
	// (returning gxpage.Pager) still satisfies registry.ServiceDiscovery.
	var sd registry.ServiceDiscovery = externalServiceDiscovery{}
	assert.NotNil(t, sd)
}

func TestPagerAliasIsGxpagePager(t *testing.T) {
	// registry.Pager is an alias of gxpage.Pager, so a page produced by the
	// registry package can be assigned to a gxpage.Pager variable.
	var p gxpage.Pager = registry.NewPage(0, 10, []any{1, 2, 3}, 10) //nolint:staticcheck
	assert.NotNil(t, p)
}

func TestShutdownConfigExportedFieldsSourceCompat(t *testing.T) {
	// Released callers access these exported atomic fields directly with value
	// semantics; changing their static type would break their source.
	cfg := &global.ShutdownConfig{}
	cfg.RejectRequest.Store(true)
	cfg.ConsumerActiveCount.Inc()
	cfg.ProviderActiveCount.Inc()
	cfg.ProviderLastReceivedRequestTime.Store(time.Now())
	cfg.Closing.Store(true)

	assert.True(t, cfg.RejectRequest.Load())
	assert.Equal(t, int32(1), cfg.ConsumerActiveCount.Load())
	assert.Equal(t, int32(1), cfg.ProviderActiveCount.Load())
	assert.False(t, cfg.ProviderLastReceivedRequestTime.Load().IsZero())
	assert.True(t, cfg.Closing.Load())
}

func TestBaseClusterInvokerDestroyedSourceCompat(t *testing.T) {
	// Released callers build the invoker with the old uber atomic field type.
	invoker := &base.BaseClusterInvoker{Destroyed: new(uberatomic.Bool)}
	assert.False(t, invoker.Destroyed.Load())
	invoker.Destroyed.Store(true)
	assert.True(t, invoker.Destroyed.Load())
}
