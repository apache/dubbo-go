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

package client

import (
	commonCfg "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// these functions are used to resolve circular dependencies temporarily.
// please refer to issue(https://github.com/apache/dubbo-go/issues/2377)
// todo(DMwangnima): remove these functions when refactoring dubbo-go
func compatApplicationConfig(c *commonCfg.ApplicationConfig) *config.ApplicationConfig {
	return &config.ApplicationConfig{
		Organization: c.Organization,
		Name:         c.Name,
		Module:       c.Module,
		Group:        c.Group,
		Version:      c.Version,
		Owner:        c.Owner,
		Environment:  c.Environment,
		MetadataType: c.MetadataType,
		Tag:          c.Tag,
	}
}

func compatRegistryConfig(c *registry.RegistryConfig) *config.RegistryConfig {
	return &config.RegistryConfig{
		Protocol:          c.Protocol,
		Timeout:           c.Timeout,
		Group:             c.Group,
		Namespace:         c.Namespace,
		TTL:               c.TTL,
		Address:           c.Address,
		Username:          c.Username,
		Password:          c.Password,
		Simplified:        c.Simplified,
		Preferred:         c.Preferred,
		Zone:              c.Zone,
		Weight:            c.Weight,
		Params:            c.Params,
		RegistryType:      c.RegistryType,
		UseAsMetaReport:   c.UseAsMetaReport,
		UseAsConfigCenter: c.UseAsConfigCenter,
	}
}
