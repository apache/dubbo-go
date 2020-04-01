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
package inmemory

import (
	"sync"
)

import (
	"github.com/apache/dubbo-go/common/logger"
	"github.com/emirpasic/gods/sets"
	"github.com/emirpasic/gods/sets/treeset"
	"github.com/emirpasic/gods/utils"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/metadata/definition"
	"github.com/apache/dubbo-go/metadata/service"
)

// InMemoryMetadataService is store and query the metadata info in memory when each service registry
type MetadataService struct {
	service.BaseMetadataService
	exportedServiceURLs   *sync.Map
	subscribedServiceURLs *sync.Map
	lock                  *sync.RWMutex
}

// NewMetadataService: initiate a metadata service
func NewMetadataService() *MetadataService {
	return &MetadataService{
		exportedServiceURLs:   new(sync.Map),
		subscribedServiceURLs: new(sync.Map),
		lock:                  new(sync.RWMutex),
	}
}

// urlComparator: defined as utils.Comparator for treeset to compare the URL
func urlComparator(a, b interface{}) int {
	url1 := a.(*common.URL)
	url2 := b.(*common.URL)
	switch {
	case url1.String() > url2.String():
		return 1
	case url1.String() < url2.String():
		return -1
	default:
		return 0
	}
}

// addURL: add URL in memory
func (mts *MetadataService) addURL(targetMap *sync.Map, url *common.URL) bool {
	var (
		urlSet interface{}
		loaded bool
	)
	logger.Debug(url.ServiceKey())
	if urlSet, loaded = targetMap.LoadOrStore(url.ServiceKey(), treeset.NewWith(urlComparator)); loaded {
		mts.lock.RLock()
		if urlSet.(*treeset.Set).Contains(url) {
			mts.lock.RUnlock()
			return false
		}
		mts.lock.RUnlock()
	}
	mts.lock.Lock()
	//double chk
	if urlSet.(*treeset.Set).Contains(url) {
		mts.lock.Unlock()
		return false
	}
	urlSet.(*treeset.Set).Add(url)
	mts.lock.Unlock()
	return true
}

// removeURL: used to remove specified url
func (mts *MetadataService) removeURL(targetMap *sync.Map, url *common.URL) {
	if value, loaded := targetMap.Load(url.ServiceKey()); loaded {
		mts.lock.Lock()
		value.(*treeset.Set).Remove(url)
		mts.lock.Unlock()
		mts.lock.RLock()
		defer mts.lock.RUnlock()
		if value.(*treeset.Set).Empty() {
			targetMap.Delete(url.ServiceKey())
		}
	}
}

// getAllService: return all the exportedUrlString except for metadataService
func (mts *MetadataService) getAllService(services *sync.Map) sets.Set {
	sets := treeset.NewWith(utils.StringComparator)
	services.Range(func(key, value interface{}) bool {
		urls := value.(*treeset.Set)
		urls.Each(func(index int, value interface{}) {
			url := value.(*common.URL)
			if url.GetParam(constant.INTERFACE_KEY, url.Path) != "MetadataService" {
				sets.Add(url.String())
			}
		})
		return true
	})
	return sets
}

// getSpecifiedService: return specified service url by serviceKey
func (mts *MetadataService) getSpecifiedService(services *sync.Map, serviceKey string, protocol string) sets.Set {
	targetSets := treeset.NewWith(utils.StringComparator)
	serviceSet, loaded := services.Load(serviceKey)
	if loaded {
		serviceSet.(*treeset.Set).Each(func(index int, value interface{}) {
			url := value.(*common.URL)
			if len(protocol) == 0 || url.Protocol == protocol || url.GetParam(constant.PROTOCOL_KEY, "") == protocol {
				targetSets.Add(value.(*common.URL).String())
			}
		})
	}
	return targetSets
}

// ExportURL: store the in memory treeset
func (mts *MetadataService) ExportURL(url common.URL) bool {
	return mts.addURL(mts.exportedServiceURLs, &url)
}

// UnexportURL: remove the url store in memory treeset
func (mts *MetadataService) UnexportURL(url common.URL) {
	mts.removeURL(mts.exportedServiceURLs, &url)
}

// SubscribeURL...
func (mts *MetadataService) SubscribeURL(url common.URL) bool {
	return mts.addURL(mts.subscribedServiceURLs, &url)
}

// UnsubscribeURL...
func (mts *MetadataService) UnsubscribeURL(url common.URL) {
	mts.removeURL(mts.subscribedServiceURLs, &url)
}

// PublishServiceDefinition: publish url's service metadata info, and write into memory
func (MetadataService) PublishServiceDefinition(url common.URL) {
	interfaceName := url.GetParam(constant.INTERFACE_KEY, "")
	isGeneric := url.GetParamBool(constant.GENERIC_KEY, false)
	if len(interfaceName) > 0 && !isGeneric {
		//judge is consumer or provider
		role := url.GetParam(constant.SIDE_KEY, "")
		//var service common.RPCService
		if role == common.RoleType(common.CONSUMER).Role() {

			//TODO:BOSS FANG
		} else if role == common.RoleType(common.PROVIDER).Role() {
			//TODO:BOSS FANG
		}

	}
}

// GetExportedURLs get all exported urls
func (mts *MetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) sets.Set {
	if serviceInterface == constant.ANY_VALUE {
		return mts.getAllService(mts.exportedServiceURLs)
	} else {
		serviceKey := definition.ServiceDescriperBuild(serviceInterface, group, version)
		return mts.getSpecifiedService(mts.exportedServiceURLs, serviceKey, protocol)
	}
}

// GetSubscribedURLs get all subscribedUrl
func (mts *MetadataService) GetSubscribedURLs() sets.Set {
	return mts.getAllService(mts.subscribedServiceURLs)
}

func (MetadataService) GetServiceDefinition(interfaceName string, group string, version string) string {
	panic("implement me")
}

func (MetadataService) GetServiceDefinitionByServiceKey(serviceKey string) string {
	panic("implement me")
}
