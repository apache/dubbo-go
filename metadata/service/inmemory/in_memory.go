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
	"github.com/emirpasic/gods/sets"
	"github.com/emirpasic/gods/sets/treeset"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/metadata/service"
)

// InMemoryMetadataService is store and query the metadata info in memory when each service registry
type MetadataService struct {
	service.BaseMetadataService
	exportedServiceURLs   sync.Map
	subscribedServiceURLs sync.Map
}

// urlComparator: defined as utils.Comparator for treeset to compare the URL
func urlComparator(a, b interface{}) int {
	url1 := a.(common.URL)
	url2 := b.(common.URL)
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
func addURL(targetMap sync.Map, url common.URL) bool {
	var (
		urlSet interface{}
		loaded bool
		lock   sync.RWMutex
	)
	if urlSet, loaded = targetMap.LoadOrStore(url.ServiceKey(), treeset.NewWith(urlComparator)); loaded {
		lock.RLock()
		if urlSet.(treeset.Set).Contains(url) {
			lock.RUnlock()
			return false
		}
		lock.RUnlock()
	}
	lock.Lock()
	urlSet.(treeset.Set).Add(url)
	lock.Unlock()
	return true
}

// name...
func removeURL(targetMap sync.Map, url common.URL) string {
	if value, loaded := targetMap.Load(url.ServiceKey()); loaded {
		value.(treeset.Set).Remove(url)
		if value.(treeset.Set).Empty() {
			targetMap.Delete(url.ServiceKey())
		}
	}
}

// ExportURL: store the in memory treeset
func (mts *MetadataService) ExportURL(url common.URL) bool {
	return addURL(mts.exportedServiceURLs, url)
}

func (MetadataService) UnexportURL(url common.URL) bool {
	panic("implement me")
}

func (mts *MetadataService) SubscribeURL(url common.URL) bool {
	return addURL(mts.subscribedServiceURLs, url)
}

func (MetadataService) UnsubscribeURL(url common.URL) bool {
	panic("implement me")
}

func (MetadataService) PublishServiceDefinition(url common.URL) {
	panic("implement me")
}

func (MetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) sets.Set {
	panic("implement me")
}

func (MetadataService) GetServiceDefinition(interfaceName string, version string, group string) string {
	panic("implement me")
}

func (MetadataService) GetServiceDefinitionByServiceKey(serviceKey string) string {
	panic("implement me")
}
