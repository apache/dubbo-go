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

package memory

import (
	"sync"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func init() {
	extension.SetGlobalServiceNameMapping(GetNameMappingInstance)
}

type InMemoryServiceNameMapping struct{}

func (i *InMemoryServiceNameMapping) Map(url *common.URL) error {
	return nil
}

func (i *InMemoryServiceNameMapping) Get(url *common.URL, listener registry.MappingListener) (*gxset.HashSet, error) {
	return gxset.NewSet(config.GetApplicationConfig().Name), nil
}

func (i *InMemoryServiceNameMapping) Remove(url *common.URL) error {
	return nil
}

var (
	serviceNameMappingInstance *InMemoryServiceNameMapping
	serviceNameMappingOnce     sync.Once
)

func GetNameMappingInstance() mapping.ServiceNameMapping {
	serviceNameMappingOnce.Do(func() {
		serviceNameMappingInstance = &InMemoryServiceNameMapping{}
	})
	return serviceNameMappingInstance
}
