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

package server

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// these functions are used to resolve circular dependencies temporarily.
// please refer to issue(https://github.com/apache/dubbo-go/issues/2377)
// todo(DMwangnima): remove these functions when refactoring dubbo-go
func compatApplicationConfig(c *global.ApplicationConfig) *config.ApplicationConfig {
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

func compatRegistryConfig(c *global.RegistryConfig) *config.RegistryConfig {
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

func compatMethodConfig(c *global.MethodConfig) *config.MethodConfig {
	return &config.MethodConfig{
		InterfaceId:                 c.InterfaceId,
		InterfaceName:               c.InterfaceName,
		Name:                        c.Name,
		Retries:                     c.Retries,
		LoadBalance:                 c.LoadBalance,
		Weight:                      c.Weight,
		TpsLimitInterval:            c.TpsLimitInterval,
		TpsLimitRate:                c.TpsLimitRate,
		TpsLimitStrategy:            c.TpsLimitStrategy,
		ExecuteLimit:                c.ExecuteLimit,
		ExecuteLimitRejectedHandler: c.ExecuteLimitRejectedHandler,
		Sticky:                      c.Sticky,
		RequestTimeout:              c.RequestTimeout,
	}
}

func compatProtocolConfig(c *global.ProtocolConfig) *config.ProtocolConfig {
	return &config.ProtocolConfig{
		Name:                 c.Name,
		Ip:                   c.Ip,
		Port:                 c.Port,
		Params:               c.Params,
		MaxServerSendMsgSize: c.MaxServerSendMsgSize,
		MaxServerRecvMsgSize: c.MaxServerRecvMsgSize,
	}
}

func init() {
	config.NewInfoInvoker = compatNewInfoInvoker
}

// these functions are used to resolve circular dependencies temporarily.
func compatNewInfoInvoker(url *common.URL, info interface{}, svc common.RPCService) protocol.Invoker {
	return newInfoInvoker(url, info.(*ServiceInfo), svc)
}
