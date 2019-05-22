// Copyright 2016-2019 hxmhlt
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"context"
	"net/url"
	"strconv"
	"time"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster/directory"
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/common/proxy"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type ReferenceConfig struct {
	context       context.Context
	pxy           *proxy.Proxy
	InterfaceName string           `required:"true"  yaml:"interface"  json:"interface,omitempty"`
	Check         *bool            `yaml:"check"  json:"check,omitempty"`
	Filter        string           `yaml:"filter" json:"filter,omitempty"`
	Protocol      string           `yaml:"protocol"  json:"protocol,omitempty"`
	Registries    []ConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	Cluster       string           `yaml:"cluster"  json:"cluster,omitempty"`
	Loadbalance   string           `yaml:"loadbalance"  json:"loadbalance,omitempty"`
	Retries       int64            `yaml:"retries"  json:"retries,omitempty"`
	Group         string           `yaml:"group"  json:"group,omitempty"`
	Version       string           `yaml:"version"  json:"version,omitempty"`
	Methods       []struct {
		Name        string `yaml:"name"  json:"name,omitempty"`
		Retries     int64  `yaml:"retries"  json:"retries,omitempty"`
		Loadbalance string `yaml:"loadbalance"  json:"loadbalance,omitempty"`
	} `yaml:"methods"  json:"methods,omitempty"`
	async   bool `yaml:"async"  json:"async,omitempty"`
	invoker protocol.Invoker
}

type ConfigRegistry string

func NewReferenceConfig(ctx context.Context) *ReferenceConfig {
	return &ReferenceConfig{context: ctx}
}
func (refconfig *ReferenceConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rf ReferenceConfig
	raw := rf{} // Put your defaults here
	if err := unmarshal(&raw); err != nil {
		return err
	}

	*refconfig = ReferenceConfig(raw)
	return nil
}
func (refconfig *ReferenceConfig) Refer() {
	//1. user specified SubURL, could be peer-to-peer address, or register center's address.

	//2. assemble SubURL from register center's configuration模式
	regUrls := loadRegistries(refconfig.Registries, consumerConfig.Registries, common.CONSUMER)
	url := common.NewURLWithOptions(refconfig.InterfaceName, common.WithProtocol(refconfig.Protocol), common.WithParams(refconfig.getUrlMap()))

	//set url to regUrls
	for _, regUrl := range regUrls {
		regUrl.SubURL = url
	}

	if len(regUrls) == 1 {
		refconfig.invoker = extension.GetProtocol("registry").Refer(*regUrls[0])

	} else {
		invokers := []protocol.Invoker{}
		for _, regUrl := range regUrls {
			invokers = append(invokers, extension.GetProtocol("registry").Refer(*regUrl))
		}
		cluster := extension.GetCluster("registryAware")
		refconfig.invoker = cluster.Join(directory.NewStaticDirectory(invokers))
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
	urlMap.Set(constant.INTERFACE_KEY, refconfig.InterfaceName)
	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, refconfig.Cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, refconfig.Loadbalance)
	urlMap.Set(constant.RETRIES_KEY, strconv.FormatInt(refconfig.Retries, 10))
	urlMap.Set(constant.GROUP_KEY, refconfig.Group)
	urlMap.Set(constant.VERSION_KEY, refconfig.Version)
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
	urlMap.Set(constant.REFERENCE_FILTER_KEY, mergeValue(consumerConfig.Filter, refconfig.Filter, constant.DEFAULT_REFERENCE_FILTERS))

	for _, v := range refconfig.Methods {
		urlMap.Set("methods."+v.Name+"."+constant.LOADBALANCE_KEY, v.Loadbalance)
		urlMap.Set("methods."+v.Name+"."+constant.RETRIES_KEY, strconv.FormatInt(v.Retries, 10))
	}

	return urlMap

}
