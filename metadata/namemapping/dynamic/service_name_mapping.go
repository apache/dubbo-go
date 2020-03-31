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
	"time"
)

import (
	"github.com/dubbogo/gost/container/set"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/metadata"
)

const (
	defaultGroup = config_center.DEFAULT_GROUP
	slash        = "/"

	// metadata service is the admin service, should not be mapped
	metadataService = "org.apache.dubbo.metadata.MetadataService"
)

// DynamicConfigurationServiceNameMapping is the implementation based on config center
type DynamicConfigurationServiceNameMapping struct {
	dc config_center.DynamicConfiguration
}

// Map will map the service to this application-level service
func (d *DynamicConfigurationServiceNameMapping) Map(serviceInterface string, group string, version string, protocol string) error {
	if metadataService == serviceInterface {
		return perrors.New("try to map the metadata service, will be ignored")
	}

	appName := config.GetApplicationConfig().Name
	value := time.Now().UnixNano()

	err := d.dc.PublishConfig(appName,
		d.buildGroup(serviceInterface, group, version, protocol),
		strconv.FormatInt(value, 10))
	if err != nil {
		return perrors.WithStack(err)
	}
	return nil
}

// Get will return the application-level services. If not found, the empty set will be returned.
// if the dynamic configuration got error, the error will return
func (d *DynamicConfigurationServiceNameMapping) Get(serviceInterface string, group string, version string, protocol string) (*gxset.HashSet, error) {
	return d.dc.GetConfigKeysByGroup(d.buildGroup(serviceInterface, group, version, protocol))
}

// buildGroup will return group, now it looks like defaultGroup/serviceInterface
func (d *DynamicConfigurationServiceNameMapping) buildGroup(
	serviceInterface string,
	group string,
	version string,
	protocol string) string {
	// the issue : https://github.com/apache/dubbo/issues/4671
	// so other params are ignored
	return defaultGroup + slash + serviceInterface
}

// NewServiceNameMapping will create an instance of DynamicConfigurationServiceNameMapping
func NewServiceNameMapping(dc config_center.DynamicConfiguration) metadata.ServiceNameMapping {
	return &DynamicConfigurationServiceNameMapping{dc: dc}
}
