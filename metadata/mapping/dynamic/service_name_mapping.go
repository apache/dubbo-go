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

package dynamic

import (
	"strconv"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/container/set"
	perrors "github.com/pkg/errors"
)

import (
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
)

const (
	defaultGroup = "mapping"
	slash        = "/"
)

func init() {
	extension.SetGlobalServiceNameMapping(GetNameMappingInstance)
}

// DynamicConfigurationServiceNameMapping is the implementation based on config center
// it's a singleton
type DynamicConfigurationServiceNameMapping struct {
	dc config_center.DynamicConfiguration
}

// Map will map the service to this application-level service
func (d *DynamicConfigurationServiceNameMapping) Map(serviceInterface string, group string, version string, protocol string) error {
	// metadata service is admin service, should not be mapped
	if constant.METADATA_SERVICE_NAME == serviceInterface {
		logger.Info("try to map the metadata service, will be ignored")
		return nil
	}

	appName := config.GetApplicationConfig().Name
	value := time.Now().UnixNano()

	err := d.dc.PublishConfig(appName,
		d.buildGroup(serviceInterface),
		strconv.FormatInt(value, 10))
	if err != nil {
		return perrors.WithStack(err)
	}
	return nil
}

// Get will return the application-level services. If not found, the empty set will be returned.
// if the dynamic configuration got error, the error will return
func (d *DynamicConfigurationServiceNameMapping) Get(serviceInterface string, group string, version string, protocol string) (*gxset.HashSet, error) {
	return d.dc.GetConfigKeysByGroup(d.buildGroup(serviceInterface))
}

// buildGroup will return group, now it looks like defaultGroup/serviceInterface
func (d *DynamicConfigurationServiceNameMapping) buildGroup(serviceInterface string) string {
	// the issue : https://github.com/apache/dubbo/issues/4671
	// so other params are ignored and remove, including group string, version string, protocol string
	return defaultGroup + slash + serviceInterface
}

var (
	serviceNameMappingInstance *DynamicConfigurationServiceNameMapping
	serviceNameMappingOnce     sync.Once
)

// GetNameMappingInstance return an instance, if not found, it creates one
func GetNameMappingInstance() mapping.ServiceNameMapping {
	serviceNameMappingOnce.Do(func() {
		dc := commonCfg.GetEnvInstance().GetDynamicConfiguration()
		serviceNameMappingInstance = &DynamicConfigurationServiceNameMapping{dc: dc}
	})
	return serviceNameMappingInstance
}
