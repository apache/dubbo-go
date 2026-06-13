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
	"encoding/gob"
	"reflect"
	"sync"
	"time"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/gof/observer"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/registry/servicediscovery/store"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

var (
	metaCache *store.CacheManager
	cacheOnce sync.Once
)

func initCache(app string) {
	gob.Register(&info.MetadataInfo{})
	fileName := constant.DefaultMetaFileName + app
	cache, err := store.NewCacheManager(constant.DefaultMetaCacheName, fileName, time.Minute*10, constant.DefaultEntrySize, true)
	if err != nil {
		logger.Fatalf("[Registry][ServiceDiscovery] failed to create cache [%s],the err is %v", constant.DefaultMetaCacheName, err)
	}
	metaCache = cache
}

// ServiceInstancesChangedListenerImpl The Service Discovery Changed  Event Listener
type ServiceInstancesChangedListenerImpl struct {
	app                string
	registryId         string
	serviceNames       *gxset.HashSet
	listeners          map[string]registry.NotifyListener
	serviceUrls        map[string][]*common.URL
	revisionToMetadata map[string]*info.MetadataInfo
	allInstances       map[string][]registry.ServiceInstance
	mutex              sync.Mutex
}

func NewServiceInstancesChangedListener(app string, registryId string, services *gxset.HashSet) registry.ServiceInstancesChangedListener {
	cacheOnce.Do(func() {
		initCache(app)
	})
	return &ServiceInstancesChangedListenerImpl{
		app:                app,
		registryId:         registryId,
		serviceNames:       services,
		listeners:          make(map[string]registry.NotifyListener),
		serviceUrls:        make(map[string][]*common.URL),
		revisionToMetadata: make(map[string]*info.MetadataInfo),
		allInstances:       make(map[string][]registry.ServiceInstance),
	}
}

// OnEvent handles service instance change events by refreshing metadata, rebuilding service URLs, and notifying listeners.
func (lstn *ServiceInstancesChangedListenerImpl) OnEvent(e observer.Event) error {
	ce, ok := e.(*registry.ServiceInstancesChangedEvent)
	if !ok {
		return nil
	}

	lstn.mutex.Lock()
	defer lstn.mutex.Unlock()

	lstn.allInstances[ce.ServiceName] = ce.Instances
	revisionToInstances := make(map[string][]registry.ServiceInstance)
	newRevisionToMetadata := make(map[string]*info.MetadataInfo)
	// The same service match key can be exported by several revisions.
	// Keep each revision's ServiceInfo so provider-specific params are not collapsed.
	serviceToRevisionServices := make(map[string]map[string]*info.ServiceInfo)
	newServiceURLs := make(map[string][]*common.URL)

	logger.Infof("[Registry][ServiceDiscovery] received instance notification event, service=%s size=%d", ce.ServiceName, len(ce.Instances))

	for _, instances := range lstn.allInstances {
		for _, instance := range instances {
			if instance.GetMetadata() == nil {
				logger.Warnf("[Registry][ServiceDiscovery] instance metadata is nil, host=%s", instance.GetHost())
				continue
			}
			revision := instance.GetMetadata()[constant.ExportedServicesRevisionPropertyName]
			if revision == "0" {
				logger.Infof("[Registry][ServiceDiscovery] find instance without valid service metadata, host=%s", instance.GetHost())
				continue
			}
			// MetadataInfo belongs to the provider application, so isolate every cache
			// dimension by provider app. Two provider apps that happen to share a revision
			// (e.g. same interface set exported under different application names) must not
			// collide on a revision-only key. instance.GetServiceName() is the provider app.
			providerApp := instance.GetServiceName()
			key := metadataCacheKey(providerApp, lstn.registryId, revision)

			subInstances := revisionToInstances[key]
			if subInstances == nil {
				subInstances = make([]registry.ServiceInstance, 0, 8)
			}
			revisionToInstances[key] = append(subInstances, instance)
			metadataInfo := lstn.revisionToMetadata[key]
			if metadataInfo == nil {
				meta, err := GetMetadataInfo(providerApp, instance, revision, lstn.registryId)
				if err != nil {
					// Skip this instance if metadata fetch fails (e.g., old Java Dubbo version)
					// Try next instance with same revision
					logger.Warnf("[Registry][ServiceDiscovery] failed to get metadata from instance %s (revision %s), err=%v, skipping this instance",
						instance.GetHost(), revision, err)
					continue
				}
				metadataInfo = meta
			}
			if metadataInfo == nil {
				logger.Warnf("[Registry][ServiceDiscovery] metadata info is nil for instance %s (revision %s), skipping this instance",
					instance.GetHost(), revision)
				continue
			}
			instance.SetServiceMetadata(metadataInfo)
			for _, service := range metadataInfo.GetServices() {
				matchKey := service.GetMatchKey()
				if serviceToRevisionServices[matchKey] == nil {
					serviceToRevisionServices[matchKey] = make(map[string]*info.ServiceInfo)
				}
				serviceToRevisionServices[matchKey][key] = service
			}

			newRevisionToMetadata[key] = metadataInfo
		}
	}
	lstn.revisionToMetadata = newRevisionToMetadata
	for key, metadataInfo := range newRevisionToMetadata {
		// key is already provider-app scoped and matches the disk cache key format.
		metaCache.Set(key, metadataInfo)
	}

	for serviceKey, revisionServices := range serviceToRevisionServices {
		urls := make([]*common.URL, 0, 8)
		for key, serviceInfo := range revisionServices {
			for _, i := range revisionToInstances[key] {
				if i != nil {
					urls = append(urls, toInstanceServiceURLs(i, serviceInfo)...)
				}
			}
		}
		newServiceURLs[serviceKey] = urls
	}

	lstn.serviceUrls = newServiceURLs
	for key, notifyListener := range lstn.listeners {
		urls := lstn.serviceUrls[key]
		events := make([]*registry.ServiceEvent, 0, len(urls))
		for _, url := range urls {
			events = append(events, &registry.ServiceEvent{
				Action:  remoting.EventTypeAdd,
				Service: url,
			})
		}
		notifyListener.NotifyAll(events, func() {})
	}

	return nil
}

func toInstanceServiceURLs(instance registry.ServiceInstance, serviceInfo *info.ServiceInfo) []*common.URL {
	urls := instance.ToURLs(serviceInfo)
	// Environment is instance-level routing metadata and is not part of the revision hash.
	// Treat the fresh instance value as authoritative so same-revision restarts
	// can update or clear stale metadata cached by revision.
	metadata := instance.GetMetadata()
	if metadata == nil {
		for _, url := range urls {
			url.DelParam(constant.EnvironmentKey)
		}
		return urls
	}
	environment, ok := metadata[constant.EnvironmentKey]
	for _, url := range urls {
		if ok && len(environment) > 0 {
			url.SetParam(constant.EnvironmentKey, environment)
		} else {
			url.DelParam(constant.EnvironmentKey)
		}
	}
	return urls
}

// AddListenerAndNotify add notify listener and notify to listen service event
func (lstn *ServiceInstancesChangedListenerImpl) AddListenerAndNotify(serviceKey string, notify registry.NotifyListener) {
	lstn.mutex.Lock()
	lstn.listeners[serviceKey] = notify
	urls := lstn.serviceUrls[serviceKey]
	lstn.mutex.Unlock()

	for _, url := range urls {
		notify.Notify(&registry.ServiceEvent{
			Action:  remoting.EventTypeAdd,
			Service: url,
		})
	}
}

// RemoveListener remove notify listener
func (lstn *ServiceInstancesChangedListenerImpl) RemoveListener(serviceKey string) {
	lstn.mutex.Lock()
	defer lstn.mutex.Unlock()
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

// metadataCacheKey builds the cache key that isolates MetadataInfo by provider
// application, registry, and revision. MetadataInfo is owned by the provider app,
// so app must be the provider application name (instance.GetServiceName()), never
// the subscribing consumer app. Keying on revision alone would let two provider
// apps that share a revision overwrite each other's metadata.
func metadataCacheKey(app, registryId, revision string) string {
	return app + ":" + registryId + ":" + revision
}

// GetMetadataInfo retrieves the MetadataInfo for a service instance by revision.
// Results are cached by app+registryId+revision, where app must be the provider
// application name. For "remote" storage type, it fetches from the metadata report
// and falls back to RPC if the report fails or returns nil. For all other storage
// types (including absent), it uses RPC directly.
func GetMetadataInfo(app string, instance registry.ServiceInstance, revision string, registryId string) (*info.MetadataInfo, error) {
	cacheOnce.Do(func() {
		initCache(app)
	})
	cacheKey := metadataCacheKey(app, registryId, revision)
	if metadataInfo, ok := metaCache.Get(cacheKey); ok {
		return metadataInfo.(*info.MetadataInfo), nil
	}

	var metadataStorageType string
	var metadataInfo *info.MetadataInfo
	var err error
	if instance.GetMetadata() == nil {
		// No metadata map at all; treat as default (local/RPC) storage type.
		metadataStorageType = constant.DefaultMetadataStorageType
	} else {
		metadataStorageType = instance.GetMetadata()[constant.MetadataStorageTypePropertyName]
		if metadataStorageType == "" {
			// MetadataStorageTypePropertyName absent (e.g. old Java provider); default to local storage type.
			logger.Warnf("[Metadata] MetadataStorageType not set for instance %s, defaulting to local", instance.GetID())
			metadataStorageType = constant.DefaultMetadataStorageType
		}
	}

	if metadataStorageType == constant.RemoteMetadataStorageType {
		var reportErr error
		metadataInfo, reportErr = metadata.GetMetadataFromMetadataReport(revision, instance, registryId)
		if reportErr != nil {
			logger.Errorf("[Metadata][Fallback] report failed, fallback to RPC app=%s registry=%s revision=%s err=%v",
				app, registryId, revision, reportErr)
		} else if metadataInfo == nil {
			logger.Warnf("[Metadata][Fallback] report returned nil metadata, fallback to RPC app=%s registry=%s revision=%s",
				app, registryId, revision)
		} else {
			metaCache.Set(cacheKey, metadataInfo)
			return metadataInfo, nil
		}

		metadataInfo, err = metadata.GetMetadataFromRpc(revision, instance)
		if err != nil {
			if reportErr != nil {
				// Wrap rpcErr so callers can use errors.Is/As on the primary failure;
				// reportErr is annotated as context since it triggered the fallback.
				return nil, perrors.Wrapf(err,
					"both paths failed, reportErr: %v", reportErr)
			}
			// reportErr was nil — the report returned nil metadata and RPC also failed.
			return nil, perrors.Wrapf(err,
				"RPC fallback failed after report returned nil metadata")
		}
		if metadataInfo == nil {
			return nil, perrors.Errorf("got nil metadata from RPC app=%s registry=%s revision=%s",
				app, registryId, revision)
		}
		metaCache.Set(cacheKey, metadataInfo)
		return metadataInfo, nil
	}

	// Non-remote storage type ("local" or absent): fetch metadata via RPC directly.
	metadataInfo, err = metadata.GetMetadataFromRpc(revision, instance)
	if err != nil {
		return nil, perrors.Wrapf(err,
			"failed app=%s registry=%s revision=%s", app, registryId, revision)
	}
	if metadataInfo == nil {
		return nil, perrors.Errorf("got nil metadata from RPC app=%s registry=%s revision=%s",
			app, registryId, revision)
	}
	metaCache.Set(cacheKey, metadataInfo)
	return metadataInfo, nil
}
