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

package config

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// ServiceDiscoveryConfig will be used to create
type ServiceDiscoveryConfig struct {
	// Protocol indicate which implementation will be used.
	// for example, if the Protocol is nacos, it means that we will use nacosServiceDiscovery
	Protocol string `yaml:"protocol" json:"protocol,omitempty" property:"protocol"`
	// Group, usually you don't need to config this field.
	// you can use this to do some isolation
	Group string `yaml:"group" json:"group,omitempty"`
	// RemoteRef is the reference point to RemoteConfig which will be used to create remotes instances.
	RemoteRef string `yaml:"remote_ref" json:"remote_ref,omitempty" property:"remote_ref"`
}

func (c *ServiceDiscoveryConfig) Prefix() string {
	return constant.ServiceDiscPrefix
}
