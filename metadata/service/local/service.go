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

package local

import (
	"sort"
	"sync"
)

import (
	"github.com/Workiva/go-datastructures/slice/skip"

	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/metadata/definition"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
)

func init() {
	extension.SetLocalMetadataService(constant.DefaultKey, GetLocalMetadataService)
}

// version will be used by Version func
const (
	version = "1.0.0"
)

// MetadataService is store and query the metadata info in memory when each service registry
type MetadataService struct {
	service.BaseMetadataService
	exportedServiceURLs   *sync.Map
	subscribedServiceURLs *sync.Map
	serviceDefinitions    *sync.Map
	lock                  *sync.RWMutex
	mOnce                 *sync.Once
	metadataInfo          *common.MetadataInfo
	metadataServiceURL    *common.URL
}

var (
	metadataServiceInstance *MetadataService
	metadataServiceInitOnce sync.Once
)

// GetLocalMetadataService which should be singleton initiates a metadata service
func GetLocalMetadataService() (service.MetadataService, error) {
	metadataServiceInitOnce.Do(func() {
		metadataServiceInstance = &MetadataService{
			BaseMetadataService:   service.NewBaseMetadataService(config.GetApplicationConfig().Name),
			exportedServiceURLs:   &sync.Map{},
			subscribedServiceURLs: &sync.Map{},
			serviceDefinitions:    &sync.Map{},
			lock:                  &sync.RWMutex{},
			metadataInfo:          nil,
			mOnce:                 &sync.Once{},
		}
	})
	return metadataServiceInstance, nil
}

// addURL will add URL in memory
func (mts *MetadataService) addURL(targetMap *sync.Map, url *common.URL) bool {
	var (
		urlSet interface{}
		loaded bool
	)
	logger.Debug(url.ServiceKey())
	if urlSet, loaded = targetMap.LoadOrStore(url.ServiceKey(), skip.New(uint64(0))); loaded {
		mts.lock.RLock()
		wantedUrl := urlSet.(*skip.SkipList).Get(url)
		if len(wantedUrl) > 0 && wantedUrl[0] != nil {
			mts.lock.RUnlock()
			return false
		}
		mts.lock.RUnlock()
	}
	mts.lock.Lock()
	// double chk
	wantedUrl := urlSet.(*skip.SkipList).Get(url)
	if len(wantedUrl) > 0 && wantedUrl[0] != nil {
		mts.lock.Unlock()
		return false
	}
	urlSet.(*skip.SkipList).Insert(url)
	mts.lock.Unlock()
	return true
}

// removeURL is used to remove specified url
func (mts *MetadataService) removeURL(targetMap *sync.Map, url *common.URL) {
	if value, loaded := targetMap.Load(url.ServiceKey()); loaded {
		mts.lock.Lock()
		value.(*skip.SkipList).Delete(url)
		mts.lock.Unlock()
		mts.lock.RLock()
		defer mts.lock.RUnlock()
		if value.(*skip.SkipList).Len() == 0 {
			targetMap.Delete(url.ServiceKey())
		}
	}
}

// getAllService can return all the exportedUrlString except for metadataService
func (mts *MetadataService) getAllService(services *sync.Map) []*common.URL {
	// using skip list to dedup and sorting
	var res []*common.URL
	services.Range(func(key, value interface{}) bool {
		urls := value.(*skip.SkipList)
		for i := uint64(0); i < urls.Len(); i++ {
			url := urls.ByPosition(i).(*common.URL)
			if url.Service() != constant.MetadataServiceName {
				res = append(res, url)
			}
		}
		return true
	})
	sort.Sort(common.URLSlice(res))
	return res
}

// getSpecifiedService can return specified service url by serviceKey
func (mts *MetadataService) getSpecifiedService(services *sync.Map, serviceKey string, protocol string) []*common.URL {
	var res []*common.URL
	serviceList, loaded := services.Load(serviceKey)
	if loaded {
		urls := serviceList.(*skip.SkipList)
		for i := uint64(0); i < urls.Len(); i++ {
			url := urls.ByPosition(i).(*common.URL)
			if len(protocol) == 0 || protocol == constant.AnyValue || url.Protocol == protocol || url.GetParam(constant.ProtocolKey, "") == protocol {
				res = append(res, url)
			}
		}
		sort.Stable(common.URLSlice(res))
	}
	return res
}

// ExportURL can store the in memory
func (mts *MetadataService) ExportURL(url *common.URL) (bool, error) {
	if constant.MetadataServiceName == url.GetParam(constant.InterfaceKey, "") {
		mts.metadataServiceURL = url
		return true, nil
	}
	mts.mOnce.Do(func() {
		mts.metadataInfo = common.NewMetadataInfWithApp(config.GetApplicationConfig().Name)
	})
	mts.metadataInfo.AddService(common.NewServiceInfoWithURL(url))
	return mts.addURL(mts.exportedServiceURLs, url), nil
}

// UnexportURL can remove the url store in memory
func (mts *MetadataService) UnexportURL(url *common.URL) error {
	if constant.MetadataServiceName == url.GetParam(constant.InterfaceKey, "") {
		mts.metadataServiceURL = nil
		return nil
	}
	if mts.metadataInfo != nil {
		mts.metadataInfo.RemoveService(common.NewServiceInfoWithURL(url))
	}
	mts.removeURL(mts.exportedServiceURLs, url)
	return nil
}

// SubscribeURL can store the in memory
func (mts *MetadataService) SubscribeURL(url *common.URL) (bool, error) {
	return mts.addURL(mts.subscribedServiceURLs, url), nil
}

// UnsubscribeURL can remove the url store in memory
func (mts *MetadataService) UnsubscribeURL(url *common.URL) error {
	mts.removeURL(mts.subscribedServiceURLs, url)
	return nil
}

// PublishServiceDefinition publish url's service metadata info, and write into memory
func (mts *MetadataService) PublishServiceDefinition(url *common.URL) error {
	if common.RoleType(common.CONSUMER).Role() == url.GetParam(constant.SideKey, "") {
		return nil
	}
	interfaceName := url.GetParam(constant.InterfaceKey, "")
	isGeneric := url.GetParamBool(constant.GenericKey, false)
	if len(interfaceName) > 0 && !isGeneric {
		tmpService := common.ServiceMap.GetServiceByServiceKey(url.Protocol, url.ServiceKey())
		sd := definition.BuildServiceDefinition(*tmpService, url)
		data, err := sd.ToBytes()
		if err != nil {
			logger.Errorf("publishProvider getServiceDescriptor error. providerUrl:%v , error:%v ", url, err)
			return nil
		}
		mts.serviceDefinitions.Store(url.ServiceKey(), string(data))
		return nil
	}
	logger.Errorf("publishProvider interfaceName is empty . providerUrl:%v ", url)
	return nil
}

// GetExportedURLs get all exported urls
func (mts *MetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]*common.URL, error) {
	if serviceInterface == constant.AnyValue {
		return mts.getAllService(mts.exportedServiceURLs), nil
	} else {
		serviceKey := definition.ServiceDescriperBuild(serviceInterface, group, version)
		return mts.getSpecifiedService(mts.exportedServiceURLs, serviceKey, protocol), nil
	}
}

// GetSubscribedURLs get all subscribedUrl
func (mts *MetadataService) GetSubscribedURLs() ([]*common.URL, error) {
	return mts.getAllService(mts.subscribedServiceURLs), nil
}

// GetServiceDefinition can get service definition by interfaceName, group and version
func (mts *MetadataService) GetServiceDefinition(interfaceName string, group string, version string) (string, error) {
	serviceKey := definition.ServiceDescriperBuild(interfaceName, group, version)
	v, _ := mts.serviceDefinitions.Load(serviceKey)
	return v.(string), nil
}

// GetServiceDefinitionByServiceKey can get service definition by serviceKey
func (mts *MetadataService) GetServiceDefinitionByServiceKey(serviceKey string) (string, error) {
	v, _ := mts.serviceDefinitions.Load(serviceKey)
	return v.(string), nil
}

// GetMetadataInfo can get metadata in memory
func (mts *MetadataService) GetMetadataInfo(revision string) (*common.MetadataInfo, error) {
	if revision == "" {
		return mts.metadataInfo, nil
	}
	if mts.metadataInfo.CalAndGetRevision() != revision {
		return nil, nil
	}
	return mts.metadataInfo, nil
}

// GetExportedServiceURLs get exported service urls
func (mts *MetadataService) GetExportedServiceURLs() ([]*common.URL, error) {
	return mts.getAllService(mts.exportedServiceURLs), nil
}

// RefreshMetadata will always return true because it will be implement by remote service
func (mts *MetadataService) RefreshMetadata(string, string) (bool, error) {
	return true, nil
}

// Version will return the version of metadata service
func (mts *MetadataService) Version() (string, error) {
	return version, nil
}

// GetMetadataServiceURL get url of MetadataService
func (mts *MetadataService) GetMetadataServiceURL() (*common.URL, error) {
	return mts.metadataServiceURL, nil
}

// SetMetadataServiceURL save url of MetadataService
func (mts *MetadataService) SetMetadataServiceURL(url *common.URL) error {
	mts.metadataServiceURL = url
	return nil
}
