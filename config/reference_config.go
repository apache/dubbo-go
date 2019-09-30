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
	"time"
)

import (
	"github.com/creasty/defaults"
)

import (
	"github.com/apache/dubbo-go/cluster/directory"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/proxy"
	"github.com/apache/dubbo-go/common/utils"
	"github.com/apache/dubbo-go/protocol"
)

type ReferenceConfig struct {
	context       context.Context
	pxy           *proxy.Proxy
	id            string
	InterfaceName string            `required:"true"  yaml:"interface"  json:"interface,omitempty" property:"interface"`
	Check         *bool             `yaml:"check"  json:"check,omitempty" property:"check"`
	Url           string            `yaml:"url"  json:"url,omitempty" property:"url"`
	Filter        string            `yaml:"filter" json:"filter,omitempty" property:"filter"`
	Protocol      string            `default:"dubbo"  yaml:"protocol"  json:"protocol,omitempty" property:"protocol"`
	Registry      string            `yaml:"registry"  json:"registry,omitempty"  property:"registry"`
	Cluster       string            `yaml:"cluster"  json:"cluster,omitempty" property:"cluster"`
	Loadbalance   string            `yaml:"loadbalance"  json:"loadbalance,omitempty" property:"loadbalance"`
	Retries       string            `yaml:"retries"  json:"retries,omitempty" property:"retries"`
	Group         string            `yaml:"group"  json:"group,omitempty" property:"group"`
	Version       string            `yaml:"version"  json:"version,omitempty" property:"version"`
	Methods       []*MethodConfig   `yaml:"methods"  json:"methods,omitempty" property:"methods"`
	async         bool              `yaml:"async"  json:"async,omitempty" property:"async"`
	Params        map[string]string `yaml:"params"  json:"params,omitempty" property:"params"`
	invoker       protocol.Invoker
	urls          []*common.URL
	Generic       bool `yaml:"generic"  json:"generic,omitempty" property:"generic"`
}

func (c *ReferenceConfig) Prefix() string {
	return constant.ReferenceConfigPrefix + c.InterfaceName + "."
}

// The only way to get a new ReferenceConfig
func NewReferenceConfig(id string, ctx context.Context) *ReferenceConfig {
	return &ReferenceConfig{id: id, context: ctx}
}

func (refconfig *ReferenceConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {

	type rf ReferenceConfig
	raw := rf{} // Put your defaults here
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*refconfig = ReferenceConfig(raw)
	if err := defaults.Set(refconfig); err != nil {
		return err
	}

	return nil
}

func (refconfig *ReferenceConfig) Refer() {
	url := common.NewURLWithOptions(common.WithPath(refconfig.id), common.WithProtocol(refconfig.Protocol), common.WithParams(refconfig.getUrlMap()))

	//1. user specified URL, could be peer-to-peer address, or register center's address.
	if refconfig.Url != "" {
		urlStrings := utils.RegSplit(refconfig.Url, "\\s*[;]+\\s*")
		for _, urlStr := range urlStrings {
			serviceUrl, err := common.NewURL(context.Background(), urlStr)
			if err != nil {
				panic(fmt.Sprintf("user specified URL %v refer error, error message is %v ", urlStr, err.Error()))
			}
			if serviceUrl.Protocol == constant.REGISTRY_PROTOCOL {
				serviceUrl.SubURL = url
				refconfig.urls = append(refconfig.urls, &serviceUrl)
			} else {
				if serviceUrl.Path == "" {
					serviceUrl.Path = "/" + refconfig.id
				}
				// merge url need to do
				newUrl := common.MergeUrl(&serviceUrl, url)
				refconfig.urls = append(refconfig.urls, newUrl)
			}

		}
	} else {
		//2. assemble SubURL from register center's configuration模式
		refconfig.urls = loadRegistries(refconfig.Registry, consumerConfig.Registries, common.CONSUMER)

		//set url to regUrls
		for _, regUrl := range refconfig.urls {
			regUrl.SubURL = url
		}
	}
	if len(refconfig.urls) == 1 {
		refconfig.invoker = extension.GetProtocol(refconfig.urls[0].Protocol).Refer(*refconfig.urls[0])
	} else {
		invokers := []protocol.Invoker{}
		var regUrl *common.URL
		for _, u := range refconfig.urls {
			invokers = append(invokers, extension.GetProtocol(u.Protocol).Refer(*u))
			if u.Protocol == constant.REGISTRY_PROTOCOL {
				regUrl = u
			}
		}
		if regUrl != nil {
			cluster := extension.GetCluster("registryAware")
			refconfig.invoker = cluster.Join(directory.NewStaticDirectory(invokers))
		} else {
			cluster := extension.GetCluster(refconfig.Cluster)
			refconfig.invoker = cluster.Join(directory.NewStaticDirectory(invokers))
		}
	}

	//create proxy
	refconfig.pxy = extension.GetProxyFactory(consumerConfig.ProxyFactory).GetProxy(refconfig.invoker, url)
}

// @v is service provider implemented RPCService
func (refconfig *ReferenceConfig) Implement(v common.RPCService) {
	refconfig.pxy.Implement(v)
}

func (refconfig *ReferenceConfig) GetRPCService() common.RPCService {
	return refconfig.pxy.Get()
}

func (refconfig *ReferenceConfig) getUrlMap() url.Values {
	urlMap := url.Values{}
	//first set user params
	for k, v := range refconfig.Params {
		urlMap.Set(k, v)
	}
	urlMap.Set(constant.INTERFACE_KEY, refconfig.InterfaceName)
	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, refconfig.Cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, refconfig.Loadbalance)
	urlMap.Set(constant.RETRIES_KEY, refconfig.Retries)
	urlMap.Set(constant.GROUP_KEY, refconfig.Group)
	urlMap.Set(constant.VERSION_KEY, refconfig.Version)
	urlMap.Set(constant.GENERIC_KEY, strconv.FormatBool(refconfig.Generic))
	urlMap.Set(constant.ROLE_KEY, strconv.Itoa(common.CONSUMER))
	//getty invoke async or sync
	urlMap.Set(constant.ASYNC_KEY, strconv.FormatBool(refconfig.async))

	//application info
	urlMap.Set(constant.APPLICATION_KEY, consumerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.ORGANIZATION_KEY, consumerConfig.ApplicationConfig.Organization)
	urlMap.Set(constant.NAME_KEY, consumerConfig.ApplicationConfig.Name)
	urlMap.Set(constant.MODULE_KEY, consumerConfig.ApplicationConfig.Module)
	urlMap.Set(constant.APP_VERSION_KEY, consumerConfig.ApplicationConfig.Version)
	urlMap.Set(constant.OWNER_KEY, consumerConfig.ApplicationConfig.Owner)
	urlMap.Set(constant.ENVIRONMENT_KEY, consumerConfig.ApplicationConfig.Environment)

	//filter
	var defaultReferenceFilter = constant.DEFAULT_REFERENCE_FILTERS
	if refconfig.Generic {
		defaultReferenceFilter = constant.GENERIC_REFERENCE_FILTERS + defaultReferenceFilter
	}
	urlMap.Set(constant.REFERENCE_FILTER_KEY, mergeValue(consumerConfig.Filter, refconfig.Filter, defaultReferenceFilter))

	for _, v := range refconfig.Methods {
		urlMap.Set("methods."+v.Name+"."+constant.LOADBALANCE_KEY, v.Loadbalance)
		urlMap.Set("methods."+v.Name+"."+constant.RETRIES_KEY, v.Retries)
	}

	return urlMap

}
func (refconfig *ReferenceConfig) GenericLoad(id string) {
	genericService := NewGenericService(refconfig.id)
	SetConsumerService(genericService)
	refconfig.id = id
	refconfig.Refer()
	refconfig.Implement(genericService)
	return
}
