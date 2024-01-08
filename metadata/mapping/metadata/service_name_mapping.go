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

package metadata

import (
	"sync"
)

import (
	gxset "github.com/dubbogo/gost/container/set"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report/instance"
)

const (
	DefaultGroup = "mapping"
	retryTimes   = 10
)

func init() {
	extension.SetGlobalServiceNameMapping(GetNameMappingInstance)
}

var (
	serviceNameMappingInstance *ServiceNameMapping
	serviceNameMappingOnce     sync.Once
)

// GetNameMappingInstance return an instance, if not found, it creates one
func GetNameMappingInstance() mapping.ServiceNameMapping {
	serviceNameMappingOnce.Do(func() {
		serviceNameMappingInstance = &ServiceNameMapping{}
	})
	return serviceNameMappingInstance
}

// ServiceNameMapping is the implementation based on metadata report
// it's a singleton
type ServiceNameMapping struct {
}

// Map will map the service to this application-level service
func (d *ServiceNameMapping) Map(url *common.URL) error {
	serviceInterface := url.GetParam(constant.InterfaceKey, "")
	appName := url.GetParam(constant.ApplicationKey, "")
	// url is the service url,not the registry url,this url has no registry id info,can not got where to write mapping,so write all
	// if the mapping can hold a report instance, it can write once
	metadataReports := instance.GetMetadataReports()
	if len(metadataReports) == 0 {
		return perrors.New("can not registering mapping to remote cause no metadata report instance found")
	} else {
		for _, metadataReport := range metadataReports {
			var err error
			for i := 0; i < retryTimes; i++ {
				err = metadataReport.RegisterServiceAppMapping(serviceInterface, DefaultGroup, appName)
				if err == nil {
					break
				}
			}
			if err != nil {
				logger.Errorf("Failed registering mapping to remote, &v", err)
			}
		}
	}
	return nil
}

// Get will return the application-level services. If not found, the empty set will be returned.
func (d *ServiceNameMapping) Get(url *common.URL, listener mapping.MappingListener) (*gxset.HashSet, error) {
	serviceInterface := url.GetParam(constant.InterfaceKey, "")
	metadataReport := instance.GetMetadataReport()
	if metadataReport == nil {
		return nil, perrors.New("can not get mapping in remote cause no metadata report instance found")
	}
	return metadataReport.GetServiceAppMapping(serviceInterface, DefaultGroup, listener)
}

func (d *ServiceNameMapping) Remove(url *common.URL) error {
	serviceInterface := url.GetParam(constant.InterfaceKey, "")
	metadataReport := instance.GetMetadataReport()
	if metadataReport == nil {
		return perrors.New("can not remove mapping in remote cause no metadata report instance found")
	}
	return metadataReport.RemoveServiceAppMappingListener(serviceInterface, DefaultGroup)
}
