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
	"sync"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type ServiceMappingChangedListenerImpl struct {
	oldServiceNames *gxset.HashSet
	listener        registry.NotifyListener
	registryUrl     *common.URL
	serviceUrl      *common.URL
	mappingCache    *sync.Map
	stop            int
}

const (
	ServiceMappingListenerStart = iota
	ServiceMappingListenerStop
)

func NewMappingListener(registryUrl *common.URL, serviceUrl *common.URL, oldServiceNames *gxset.HashSet, listener registry.NotifyListener) *ServiceMappingChangedListenerImpl {
	return &ServiceMappingChangedListenerImpl{
		listener:        listener,
		oldServiceNames: oldServiceNames,
		registryUrl:     registryUrl,
		serviceUrl:      serviceUrl,
		stop:            ServiceMappingListenerStart,
		mappingCache:    &sync.Map{},
	}
}

// OnEvent on ServiceMappingChangedEvent the service mapping change event
func (lstn *ServiceMappingChangedListenerImpl) OnEvent(e observer.Event) error {
	var (
		err error
		reg registry.Registry
	)
	if lstn.stop == ServiceMappingListenerStop {
		return nil
	}
	sm, ok := e.(*registry.ServiceMappingChangeEvent)
	if !ok {
		return nil
	}
	newServiceNames := sm.GetServiceNames()
	oldServiceNames := lstn.oldServiceNames
	// serviceMapping is orderly
	if newServiceNames.Empty() || oldServiceNames.String() == newServiceNames.String() {
		return nil
	}
	if newServiceNames.Size() > 0 && oldServiceNames.Empty() {
		if reg, err = extension.GetRegistry(lstn.registryUrl.Protocol, lstn.registryUrl); err != nil {
			return err
		}
		if sdreg, ok := reg.(*ServiceDiscoveryRegistry); ok {
			sdreg.SubscribeURL(lstn.serviceUrl, lstn.listener, newServiceNames)
		}
		lstn.oldServiceNames = newServiceNames
		return nil
	}
	for _, service := range newServiceNames.Values() {
		if !oldServiceNames.Contains(service) {
			lstn.mappingCache.Delete(oldServiceNames.String())
			lstn.mappingCache.Store(newServiceNames.String(), newServiceNames)
			if reg, err = extension.GetRegistry(lstn.registryUrl.Protocol, lstn.registryUrl); err != nil {
				return err
			}
			if sdreg, ok := reg.(*ServiceDiscoveryRegistry); ok {
				sdreg.SubscribeURL(lstn.serviceUrl, lstn.listener, newServiceNames)
			}
			lstn.oldServiceNames = newServiceNames
		}
	}
	return err
}

// Stop on ServiceMappingChangedEvent the service mapping change event
func (lstn *ServiceMappingChangedListenerImpl) Stop() {
	lstn.stop = ServiceMappingListenerStop
}
