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
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config/instance"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
)

const (
	defaultGroup = "mapping"
	slash        = "/"
)

func init() {
	extension.SetGlobalServiceNameMapping(GetNameMappingInstance)
}

// MetadataServiceNameMapping is the implementation based on metadata report
// it's a singleton
type MetadataServiceNameMapping struct {
}

// Map will map the service to this application-level service
func (d *MetadataServiceNameMapping) Map(url *common.URL) error {
	serviceInterface := url.GetParam(constant.InterfaceKey, "")
	// metadata service is admin service, should not be mapped
	if constant.MetadataServiceName == serviceInterface {
		logger.Debug("try to map the metadata service, will be ignored")
		return nil
	}

	appName := config.GetApplicationConfig().Name

	metadataReport := getMetaDataReport(url.GetParam(constant.RegistryKey, ""))
	if metadataReport == nil {
		return perrors.New("get metadata report instance is nil")
	}
	err := metadataReport.RegisterServiceAppMapping(serviceInterface, defaultGroup, appName)
	if err != nil {
		return perrors.WithStack(err)
	}
	return nil
}

// Get will return the application-level services. If not found, the empty set will be returned.
func (d *MetadataServiceNameMapping) Get(url *common.URL) (*gxset.HashSet, error) {
	serviceInterface := url.GetParam(constant.InterfaceKey, "")
	metadataReport := instance.GetMetadataReportInstance()
	return metadataReport.GetServiceAppMapping(serviceInterface, defaultGroup)
}

// buildMappingKey will return mapping key, it looks like defaultGroup/serviceInterface
func (d *MetadataServiceNameMapping) buildMappingKey(serviceInterface string) string {
	// the issue : https://github.com/apache/dubbo/issues/4671
	// so other params are ignored and remove, including group string, version string, protocol string
	return defaultGroup + slash + serviceInterface
}

var (
	serviceNameMappingInstance *MetadataServiceNameMapping
	serviceNameMappingOnce     sync.Once
)

// GetNameMappingInstance return an instance, if not found, it creates one
func GetNameMappingInstance() mapping.ServiceNameMapping {
	serviceNameMappingOnce.Do(func() {
		serviceNameMappingInstance = &MetadataServiceNameMapping{}
	})
	return serviceNameMappingInstance
}

// getMetaDataReport obtain metadata reporting instances through registration protocol
// if the metadata type is remote, obtain the instance from the metadata report configuration
func getMetaDataReport(protocol string) report.MetadataReport {
	var metadataReport report.MetadataReport
	if config.GetApplicationConfig().MetadataType == constant.RemoteMetadataStorageType {
		metadataReport = instance.GetMetadataReportInstance()
		return metadataReport
	}
	return instance.GetMetadataReportByRegistryProtocol(protocol)
}
