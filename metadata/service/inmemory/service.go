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
	"encoding/json"
	"sync"
)

import (
	"github.com/emirpasic/gods/sets"
	"github.com/emirpasic/gods/sets/treeset"
	"github.com/emirpasic/gods/utils"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/metadata/definition"
	"github.com/apache/dubbo-go/metadata/service"
)

// MetadataService is store and query the metadata info in memory when each service registry
type MetadataService struct {
	service.BaseMetadataService
	exportedServiceURLs   *sync.Map
	subscribedServiceURLs *sync.Map
	serviceDefinitions    *sync.Map
	lock                  *sync.RWMutex
}

// NewMetadataService: initiate a metadata service
func NewMetadataService() *MetadataService {
	return &MetadataService{
		exportedServiceURLs:   &sync.Map{},
		subscribedServiceURLs: &sync.Map{},
		serviceDefinitions:    &sync.Map{},
		lock:                  &sync.RWMutex{},
	}
}

// urlComparator is defined as utils.Comparator for treeset to compare the URL
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

// addURL will add URL in memory
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

// removeURL is used to remove specified url
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

// getAllService can return all the exportedUrlString except for metadataService
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

// getSpecifiedService can return specified service url by serviceKey
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

// ExportURL can store the in memory treeset
func (mts *MetadataService) ExportURL(url common.URL) (bool, error) {
	return mts.addURL(mts.exportedServiceURLs, &url), nil
}

// UnexportURL can remove the url store in memory treeset
func (mts *MetadataService) UnexportURL(url common.URL) error {
	mts.removeURL(mts.exportedServiceURLs, &url)
	return nil
}

// SubscribeURL can store the in memory treeset
func (mts *MetadataService) SubscribeURL(url common.URL) (bool, error) {
	return mts.addURL(mts.subscribedServiceURLs, &url), nil
}

// UnsubscribeURL can remove the url store in memory treeset
func (mts *MetadataService) UnsubscribeURL(url common.URL) error {
	mts.removeURL(mts.subscribedServiceURLs, &url)
	return nil
}

// PublishServiceDefinition: publish url's service metadata info, and write into memory
func (mts *MetadataService) PublishServiceDefinition(url common.URL) error {
	interfaceName := url.GetParam(constant.INTERFACE_KEY, "")
	isGeneric := url.GetParamBool(constant.GENERIC_KEY, false)
	if len(interfaceName) > 0 && !isGeneric {
		//judge is consumer or provider
		//side := url.GetParam(constant.SIDE_KEY, "")
		//var service common.RPCService
		service := common.ServiceMap.GetService(url.Protocol, url.GetParam(constant.BEAN_NAME_KEY, url.Service()))
		//if side == common.RoleType(common.CONSUMER).Role() {
		//	//TODO:generate the service definition and store it
		//
		//} else if side == common.RoleType(common.PROVIDER).Role() {
		//	//TODO:generate the service definition and store it
		//}
		sd := definition.BuildServiceDefinition(*service, url)
		data, err := json.Marshal(sd)
		if err != nil {
			logger.Errorf("publishProvider getServiceDescriptor error. providerUrl:%v , error: ", url, err)
		}
		mts.serviceDefinitions.Store(url.ServiceKey(), string(data))
		return nil
	}
	logger.Errorf("publishProvider interfaceName is empty . providerUrl:%v ", url)
	return nil
}

// GetExportedURLs get all exported urls
func (mts *MetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) (sets.Set, error) {
	if serviceInterface == constant.ANY_VALUE {
		return mts.getAllService(mts.exportedServiceURLs), nil
	} else {
		serviceKey := definition.ServiceDescriperBuild(serviceInterface, group, version)
		return mts.getSpecifiedService(mts.exportedServiceURLs, serviceKey, protocol), nil
	}
}

// GetSubscribedURLs get all subscribedUrl
func (mts *MetadataService) GetSubscribedURLs() (sets.Set, error) {
	return mts.getAllService(mts.subscribedServiceURLs), nil
}

// GetServiceDefinition can get service definition by interfaceName, group and version
func (mts *MetadataService) GetServiceDefinition(interfaceName string, group string, version string) (string, error) {
	serviceKey := definition.ServiceDescriperBuild(interfaceName, group, version)
	v, _ := mts.serviceDefinitions.Load(serviceKey)
	return v.(string), nil
}

// GetServiceDefinition can get service definition by serviceKey
func (mts *MetadataService) GetServiceDefinitionByServiceKey(serviceKey string) (string, error) {
	v, _ := mts.serviceDefinitions.Load(serviceKey)
	return v.(string), nil
}

// Version will return the version of metadata service
func (mts *MetadataService) Version() string {
	return "1.0.0"
}

// Version will return the version of metadata service
func (mts *MetadataService) Reference() string {
	return "MetadataService"
}
