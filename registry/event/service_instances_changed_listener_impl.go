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

package event

import (
	"reflect"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

// ServiceInstancesChangedListenerImpl The Service Discovery Changed  Event Listener
type ServiceInstancesChangedListenerImpl struct {
	serviceNames       *gxset.HashSet
	listeners          map[string]registry.NotifyListener
	serviceUrls        map[string][]*common.URL
	revisionToMetadata map[string]*common.MetadataInfo
	allInstances       map[string][]registry.ServiceInstance
}

func NewServiceInstancesChangedListener(services *gxset.HashSet) registry.ServiceInstancesChangedListener {
	return &ServiceInstancesChangedListenerImpl{
		serviceNames:       services,
		listeners:          make(map[string]registry.NotifyListener),
		serviceUrls:        make(map[string][]*common.URL),
		revisionToMetadata: make(map[string]*common.MetadataInfo),
		allInstances:       make(map[string][]registry.ServiceInstance),
	}
}

// OnEvent on ServiceInstancesChangedEvent the service instances change event
func (lstn *ServiceInstancesChangedListenerImpl) OnEvent(e observer.Event) error {
	ce, ok := e.(*registry.ServiceInstancesChangedEvent)
	if !ok {
		return nil
	}
	var err error
	lstn.allInstances[ce.ServiceName] = ce.Instances
	revisionToInstances := make(map[string][]registry.ServiceInstance)
	newRevisionToMetadata := make(map[string]*common.MetadataInfo)
	localServiceToRevisions := make(map[*common.ServiceInfo]*gxset.HashSet)
	protocolRevisionsToUrls := make(map[string]map[*gxset.HashSet][]*common.URL)
	newServiceURLs := make(map[string][]*common.URL)

	for _, instances := range lstn.allInstances {
		for _, instance := range instances {
			if instance.GetMetadata() == nil {
				logger.Warnf("Instance metadata is nil: %s", instance.GetHost())
				continue
			}
			revision := instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName]
			if "0" == revision {
				logger.Infof("Find instance without valid service metadata: %s", instance.GetHost())
				continue
			}
			subInstances := revisionToInstances[revision]
			if subInstances == nil {
				subInstances = make([]registry.ServiceInstance, 8)
			}
			revisionToInstances[revision] = append(subInstances, instance)
			metadataInfo := lstn.revisionToMetadata[revision]
			if metadataInfo == nil {
				metadataInfo, err = lstn.getMetadataInfo(instance, revision)
				if err != nil {
					return err
				}
			}
			instance.SetServiceMetadata(metadataInfo)
			for _, service := range metadataInfo.Services {
				if localServiceToRevisions[service] == nil {
					localServiceToRevisions[service] = gxset.NewSet()
				}
				localServiceToRevisions[service].Add(revision)
			}

			newRevisionToMetadata[revision] = metadataInfo
		}
		lstn.revisionToMetadata = newRevisionToMetadata

		for serviceInstance, revisions := range localServiceToRevisions {
			revisionsToUrls := protocolRevisionsToUrls[serviceInstance.Protocol]
			if revisionsToUrls == nil {
				protocolRevisionsToUrls[serviceInstance.Protocol] = make(map[*gxset.HashSet][]*common.URL)
				revisionsToUrls = protocolRevisionsToUrls[serviceInstance.Protocol]
			}
			urls := revisionsToUrls[revisions]
			if urls != nil {
				newServiceURLs[serviceInstance.GetMatchKey()] = urls
			} else {
				urls = make([]*common.URL, 0, 8)
				for _, v := range revisions.Values() {
					r := v.(string)
					for _, i := range revisionToInstances[r] {
						if i != nil {
							urls = append(urls, i.ToURLs()...)
						}
					}
				}
				revisionsToUrls[revisions] = urls
				newServiceURLs[serviceInstance.GetMatchKey()] = urls
			}
		}
		lstn.serviceUrls = newServiceURLs

		for key, notifyListener := range lstn.listeners {
			urls := lstn.serviceUrls[key]
			for _, url := range urls {
				notifyListener.Notify(&registry.ServiceEvent{
					Action:  remoting.EventTypeAdd,
					Service: url,
				})
			}
		}
	}
	return nil
}

// getMetadataInfo get metadata info when MetadataStorageTypePropertyName is null
func (lstn *ServiceInstancesChangedListenerImpl) getMetadataInfo(instance registry.ServiceInstance, revision string) (*common.MetadataInfo, error) {
	var metadataStorageType string
	var metadataInfo *common.MetadataInfo
	if instance.GetMetadata() == nil {
		metadataStorageType = constant.DefaultMetadataStorageType
	} else {
		metadataStorageType = instance.GetMetadata()[constant.MetadataStorageTypePropertyName]
	}
	if metadataStorageType == constant.RemoteMetadataStorageType {
		remoteMetadataServiceImpl, err := extension.GetRemoteMetadataService()
		if err != nil {
			return nil, err
		}
		metadataInfo, err = remoteMetadataServiceImpl.GetMetadata(instance)
		if err != nil {
			return nil, err
		}
	} else {
		var err error
		proxyFactory := extension.GetMetadataServiceProxyFactory(constant.DefaultKey)
		metadataService := proxyFactory.GetProxy(instance)
		metadataInfo, err = metadataService.GetMetadataInfo(revision)
		if err != nil {
			return nil, err
		}
	}
	return metadataInfo, nil
}

// AddListenerAndNotify add notify listener and notify to listen service event
func (lstn *ServiceInstancesChangedListenerImpl) AddListenerAndNotify(serviceKey string, notify registry.NotifyListener) {
	lstn.listeners[serviceKey] = notify
	urls := lstn.serviceUrls[serviceKey]
	for _, url := range urls {
		notify.Notify(&registry.ServiceEvent{
			Action:  remoting.EventTypeAdd,
			Service: url,
		})
	}
}

// RemoveListener remove notify listener
func (lstn *ServiceInstancesChangedListenerImpl) RemoveListener(serviceKey string) {
	delete(lstn.listeners, serviceKey)
}

// GetServiceNames return all listener service names
func (lstn *ServiceInstancesChangedListenerImpl) GetServiceNames() *gxset.HashSet {
	return lstn.serviceNames
}

// Accept return true if the name is the same
func (lstn *ServiceInstancesChangedListenerImpl) Accept(e observer.Event) bool {
	if ce, ok := e.(*registry.ServiceInstancesChangedEvent); ok {
		return lstn.serviceNames.Contains(ce.ServiceName)
	}
	return false
}

// GetPriority returns -1, it will be the first invoked listener
func (lstn *ServiceInstancesChangedListenerImpl) GetPriority() int {
	return -1
}

// GetEventType returns ServiceInstancesChangedEvent
func (lstn *ServiceInstancesChangedListenerImpl) GetEventType() reflect.Type {
	return reflect.TypeOf(&registry.ServiceInstancesChangedEvent{})
}
