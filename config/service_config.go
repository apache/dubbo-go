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

type ServiceConfig struct {
	context       context.Context
	id            string
	Filter        string            `yaml:"filter" json:"filter,omitempty" property:"filter"`
	Protocol      string            `default:"dubbo"  required:"true"  yaml:"protocol"  json:"protocol,omitempty" property:"protocol"` //multi protocol support, split by ','
	InterfaceName string            `required:"true"  yaml:"interface"  json:"interface,omitempty" property:"interface"`
	Registry      string            `yaml:"registry"  json:"registry,omitempty"  property:"registry"`
	Cluster       string            `default:"failover" yaml:"cluster"  json:"cluster,omitempty" property:"cluster"`
	Loadbalance   string            `default:"random" yaml:"loadbalance"  json:"loadbalance,omitempty"  property:"loadbalance"`
	Group         string            `yaml:"group"  json:"group,omitempty" property:"group"`
	Version       string            `yaml:"version"  json:"version,omitempty" property:"version" `
	Methods       []*MethodConfig   `yaml:"methods"  json:"methods,omitempty" property:"methods"`
	Warmup        string            `yaml:"warmup"  json:"warmup,omitempty"  property:"warmup"`
	Retries       string            `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	Params        map[string]string `yaml:"params"  json:"params,omitempty" property:"params"`
	Token         string            `yaml:"token" json:"token,omitempty" property:"token"`
	unexported    *atomic.Bool
	exported      *atomic.Bool
	rpcService    common.RPCService
	cacheProtocol protocol.Protocol
	cacheMutex    sync.Mutex
}

func (c *ServiceConfig) Prefix() string {
	return constant.ServiceConfigPrefix + c.InterfaceName + "."
}

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

// The only way to get a new ServiceConfig
func NewServiceConfig(id string, context context.Context) *ServiceConfig {

	return &ServiceConfig{
		context:    context,
		id:         id,
		unexported: atomic.NewBool(false),
		exported:   atomic.NewBool(false),
	}

}

func (srvconfig *ServiceConfig) Export() error {
	//TODO: config center start here

	//TODO:delay export
	if srvconfig.unexported != nil && srvconfig.unexported.Load() {
		err := perrors.Errorf("The service %v has already unexported! ", srvconfig.InterfaceName)
		logger.Errorf(err.Error())
		return err
	}
	if srvconfig.unexported != nil && srvconfig.exported.Load() {
		logger.Warnf("The service %v has already exported! ", srvconfig.InterfaceName)
		return nil
	}

	regUrls := loadRegistries(srvconfig.Registry, providerConfig.Registries, common.PROVIDER)
	urlMap := srvconfig.getUrlMap()

	for _, proto := range loadProtocol(srvconfig.Protocol, providerConfig.Protocols) {
		//registry the service reflect
		methods, err := common.ServiceMap.Register(proto.Name, srvconfig.rpcService)
		if err != nil {
			err := perrors.Errorf("The service %v  export the protocol %v error! Error message is %v .", srvconfig.InterfaceName, proto.Name, err.Error())
			logger.Errorf(err.Error())
			return err
		}
		url := common.NewURLWithOptions(common.WithPath(srvconfig.id),
			common.WithProtocol(proto.Name),
			common.WithIp(proto.Ip),
			common.WithPort(proto.Port),
			common.WithParams(urlMap),
			common.WithParamsValue(constant.BEAN_NAME_KEY, srvconfig.id),
			common.WithMethods(strings.Split(methods, ",")),
			common.WithToken(srvconfig.Token),
		)

		if len(regUrls) > 0 {
			for _, regUrl := range regUrls {
				regUrl.SubURL = url

				srvconfig.cacheMutex.Lock()
				if srvconfig.cacheProtocol == nil {
					logger.Infof(fmt.Sprintf("First load the registry protocol , url is {%v}!", url))
					srvconfig.cacheProtocol = extension.GetProtocol("registry")
				}
				srvconfig.cacheMutex.Unlock()

				invoker := extension.GetProxyFactory(providerConfig.ProxyFactory).GetInvoker(*regUrl)
				exporter := srvconfig.cacheProtocol.Export(invoker)
				if exporter == nil {
					panic(perrors.New(fmt.Sprintf("Registry protocol new exporter error,registry is {%v},url is {%v}", regUrl, url)))
				}
			}
		} else {
			invoker := extension.GetProxyFactory(providerConfig.ProxyFactory).GetInvoker(*url)
			exporter := extension.GetProtocol(protocolwrapper.FILTER).Export(invoker)
			if exporter == nil {
				panic(perrors.New(fmt.Sprintf("Filter protocol without registry new exporter error,url is {%v}", url)))
			}
		}

	}
	return nil

}

func (srvconfig *ServiceConfig) Implement(s common.RPCService) {
	srvconfig.rpcService = s
}

func (srvconfig *ServiceConfig) getUrlMap() url.Values {
	urlMap := url.Values{}
	//first set user params
	for k, v := range srvconfig.Params {
		urlMap.Set(k, v)
	}
	urlMap.Set(constant.INTERFACE_KEY, srvconfig.InterfaceName)
	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, srvconfig.Cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, srvconfig.Loadbalance)
	urlMap.Set(constant.WARMUP_KEY, srvconfig.Warmup)
	urlMap.Set(constant.RETRIES_KEY, srvconfig.Retries)
	urlMap.Set(constant.GROUP_KEY, srvconfig.Group)
	urlMap.Set(constant.VERSION_KEY, srvconfig.Version)
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER))
	//application info
	urlMap.Set(constant.APPLICATION_KEY, providerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.ORGANIZATION_KEY, providerConfig.ApplicationConfig.Organization)
	urlMap.Set(constant.NAME_KEY, providerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.MODULE_KEY, providerConfig.ApplicationConfig.Module)
	urlMap.Set(constant.APP_VERSION_KEY, providerConfig.ApplicationConfig.Version)
	urlMap.Set(constant.OWNER_KEY, providerConfig.ApplicationConfig.Owner)
	urlMap.Set(constant.ENVIRONMENT_KEY, providerConfig.ApplicationConfig.Environment)

	//filter
	urlMap.Set(constant.SERVICE_FILTER_KEY, mergeValue(providerConfig.Filter, srvconfig.Filter, constant.DEFAULT_SERVICE_FILTERS))

	for _, v := range srvconfig.Methods {
		urlMap.Set("methods."+v.Name+"."+constant.LOADBALANCE_KEY, v.Loadbalance)
		urlMap.Set("methods."+v.Name+"."+constant.RETRIES_KEY, v.Retries)
		urlMap.Set("methods."+v.Name+"."+constant.WEIGHT_KEY, strconv.FormatInt(v.Weight, 10))
	}

	return urlMap

}
