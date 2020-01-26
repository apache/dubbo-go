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
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/creasty/defaults"
	perrors "github.com/pkg/errors"
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/protocolwrapper"
)

// ServiceConfig ...
type ServiceConfig struct {
	context                     context.Context
	id                          string
	Filter                      string            `yaml:"filter" json:"filter,omitempty" property:"filter"`
	Protocol                    string            `default:"dubbo"  required:"true"  yaml:"protocol"  json:"protocol,omitempty" property:"protocol"` // multi protocol support, split by ','
	InterfaceName               string            `required:"true"  yaml:"interface"  json:"interface,omitempty" property:"interface"`
	Registry                    string            `yaml:"registry"  json:"registry,omitempty"  property:"registry"`
	Cluster                     string            `default:"failover" yaml:"cluster"  json:"cluster,omitempty" property:"cluster"`
	Loadbalance                 string            `default:"random" yaml:"loadbalance"  json:"loadbalance,omitempty"  property:"loadbalance"`
	Group                       string            `yaml:"group"  json:"group,omitempty" property:"group"`
	Version                     string            `yaml:"version"  json:"version,omitempty" property:"version" `
	Methods                     []*MethodConfig   `yaml:"methods"  json:"methods,omitempty" property:"methods"`
	Warmup                      string            `yaml:"warmup"  json:"warmup,omitempty"  property:"warmup"`
	Retries                     string            `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	Params                      map[string]string `yaml:"params"  json:"params,omitempty" property:"params"`
	Token                       string            `yaml:"token" json:"token,omitempty" property:"token"`
	AccessLog                   string            `yaml:"accesslog" json:"accesslog,omitempty" property:"accesslog"`
	TpsLimiter                  string            `yaml:"tps.limiter" json:"tps.limiter,omitempty" property:"tps.limiter"`
	TpsLimitInterval            string            `yaml:"tps.limit.interval" json:"tps.limit.interval,omitempty" property:"tps.limit.interval"`
	TpsLimitRate                string            `yaml:"tps.limit.rate" json:"tps.limit.rate,omitempty" property:"tps.limit.rate"`
	TpsLimitStrategy            string            `yaml:"tps.limit.strategy" json:"tps.limit.strategy,omitempty" property:"tps.limit.strategy"`
	TpsLimitRejectedHandler     string            `yaml:"tps.limit.rejected.handler" json:"tps.limit.rejected.handler,omitempty" property:"tps.limit.rejected.handler"`
	ExecuteLimit                string            `yaml:"execute.limit" json:"execute.limit,omitempty" property:"execute.limit"`
	ExecuteLimitRejectedHandler string            `yaml:"execute.limit.rejected.handler" json:"execute.limit.rejected.handler,omitempty" property:"execute.limit.rejected.handler"`

	unexported    *atomic.Bool
	exported      *atomic.Bool
	rpcService    common.RPCService
	cacheProtocol protocol.Protocol
	cacheMutex    sync.Mutex
}

// Prefix ...
func (c *ServiceConfig) Prefix() string {
	return constant.ServiceConfigPrefix + c.InterfaceName + "."
}

// UnmarshalYAML ...
func (c *ServiceConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	if err := defaults.Set(c); err != nil {
		return err
	}
	type plain ServiceConfig
	if err := unmarshal((*plain)(c)); err != nil {
		return err
	}
	return nil
}

// NewServiceConfig The only way to get a new ServiceConfig
func NewServiceConfig(id string, context context.Context) *ServiceConfig {

	return &ServiceConfig{
		context:    context,
		id:         id,
		unexported: atomic.NewBool(false),
		exported:   atomic.NewBool(false),
	}

}

// Export ...
func (c *ServiceConfig) Export() error {
	// TODO: config center start here

	// TODO:delay export
	if c.unexported != nil && c.unexported.Load() {
		err := perrors.Errorf("The service %v has already unexported! ", c.InterfaceName)
		logger.Errorf(err.Error())
		return err
	}
	if c.unexported != nil && c.exported.Load() {
		logger.Warnf("The service %v has already exported! ", c.InterfaceName)
		return nil
	}

	regUrls := loadRegistries(c.Registry, providerConfig.Registries, common.PROVIDER)
	urlMap := c.getUrlMap()
	protocolConfigs := loadProtocol(c.Protocol, providerConfig.Protocols)
	if len(protocolConfigs) == 0 {
		logger.Warnf("The service %v's '%v' protocols don't has right protocolConfigs ", c.InterfaceName, c.Protocol)
		return nil
	}
	for _, proto := range protocolConfigs {
		// registry the service reflect
		methods, err := common.ServiceMap.Register(proto.Name, c.rpcService)
		if err != nil {
			err := perrors.Errorf("The service %v  export the protocol %v error! Error message is %v .", c.InterfaceName, proto.Name, err.Error())
			logger.Errorf(err.Error())
			return err
		}
		ivkURL := common.NewURLWithOptions(
			common.WithPath(c.id),
			common.WithProtocol(proto.Name),
			common.WithIp(proto.Ip),
			common.WithPort(proto.Port),
			common.WithParams(urlMap),
			common.WithParamsValue(constant.BEAN_NAME_KEY, c.id),
			common.WithMethods(strings.Split(methods, ",")),
			common.WithToken(c.Token),
		)

		if len(regUrls) > 0 {
			for _, regUrl := range regUrls {
				regUrl.SubURL = ivkURL

				c.cacheMutex.Lock()
				if c.cacheProtocol == nil {
					logger.Infof(fmt.Sprintf("First load the registry protocol , url is {%v}!", ivkURL))
					c.cacheProtocol = extension.GetProtocol("registry")
				}
				c.cacheMutex.Unlock()

				invoker := extension.GetProxyFactory(providerConfig.ProxyFactory).GetInvoker(*regUrl)
				exporter := c.cacheProtocol.Export(invoker)
				if exporter == nil {
					panic(perrors.New(fmt.Sprintf("Registry protocol new exporter error,registry is {%v},url is {%v}", regUrl, ivkURL)))
				}
			}
		} else {
			invoker := extension.GetProxyFactory(providerConfig.ProxyFactory).GetInvoker(*ivkURL)
			exporter := extension.GetProtocol(protocolwrapper.FILTER).Export(invoker)
			if exporter == nil {
				panic(perrors.New(fmt.Sprintf("Filter protocol without registry new exporter error,url is {%v}", ivkURL)))
			}
		}

	}
	return nil

}

// Implement ...
func (c *ServiceConfig) Implement(s common.RPCService) {
	c.rpcService = s
}

func (c *ServiceConfig) getUrlMap() url.Values {
	urlMap := url.Values{}
	// first set user params
	for k, v := range c.Params {
		urlMap.Set(k, v)
	}
	urlMap.Set(constant.INTERFACE_KEY, c.InterfaceName)
	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, c.Cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, c.Loadbalance)
	urlMap.Set(constant.WARMUP_KEY, c.Warmup)
	urlMap.Set(constant.RETRIES_KEY, c.Retries)
	urlMap.Set(constant.GROUP_KEY, c.Group)
	urlMap.Set(constant.VERSION_KEY, c.Version)
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	// application info
	urlMap.Set(constant.APPLICATION_KEY, providerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.ORGANIZATION_KEY, providerConfig.ApplicationConfig.Organization)
	urlMap.Set(constant.NAME_KEY, providerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.MODULE_KEY, providerConfig.ApplicationConfig.Module)
	urlMap.Set(constant.APP_VERSION_KEY, providerConfig.ApplicationConfig.Version)
	urlMap.Set(constant.OWNER_KEY, providerConfig.ApplicationConfig.Owner)
	urlMap.Set(constant.ENVIRONMENT_KEY, providerConfig.ApplicationConfig.Environment)

	// filter
	urlMap.Set(constant.SERVICE_FILTER_KEY, mergeValue(providerConfig.Filter, c.Filter, constant.DEFAULT_SERVICE_FILTERS))

	// filter special config
	urlMap.Set(constant.ACCESS_LOG_KEY, c.AccessLog)
	// tps limiter
	urlMap.Set(constant.TPS_LIMIT_STRATEGY_KEY, c.TpsLimitStrategy)
	urlMap.Set(constant.TPS_LIMIT_INTERVAL_KEY, c.TpsLimitInterval)
	urlMap.Set(constant.TPS_LIMIT_RATE_KEY, c.TpsLimitRate)
	urlMap.Set(constant.TPS_LIMITER_KEY, c.TpsLimiter)
	urlMap.Set(constant.TPS_REJECTED_EXECUTION_HANDLER_KEY, c.TpsLimitRejectedHandler)

	// execute limit filter
	urlMap.Set(constant.EXECUTE_LIMIT_KEY, c.ExecuteLimit)
	urlMap.Set(constant.EXECUTE_REJECTED_EXECUTION_HANDLER_KEY, c.ExecuteLimitRejectedHandler)

	for _, v := range c.Methods {
		prefix := "methods." + v.Name + "."
		urlMap.Set(prefix+constant.LOADBALANCE_KEY, v.Loadbalance)
		urlMap.Set(prefix+constant.RETRIES_KEY, v.Retries)
		urlMap.Set(prefix+constant.WEIGHT_KEY, strconv.FormatInt(v.Weight, 10))

		urlMap.Set(prefix+constant.TPS_LIMIT_STRATEGY_KEY, v.TpsLimitStrategy)
		urlMap.Set(prefix+constant.TPS_LIMIT_INTERVAL_KEY, v.TpsLimitInterval)
		urlMap.Set(prefix+constant.TPS_LIMIT_RATE_KEY, v.TpsLimitRate)

		urlMap.Set(constant.EXECUTE_LIMIT_KEY, v.ExecuteLimit)
		urlMap.Set(constant.EXECUTE_REJECTED_EXECUTION_HANDLER_KEY, v.ExecuteLimitRejectedHandler)

	}

	return urlMap

}
