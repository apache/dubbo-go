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
	"strings"
	"sync"
	"time"
)
import (
	log "github.com/AlexStocks/log4go"
	perrors "github.com/pkg/errors"
	"go.uber.org/atomic"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type ServiceConfig struct {
	context       context.Context
	Filter        string           `yaml:"filter" json:"filter,omitempty"`
	Protocol      string           `required:"true"  yaml:"protocol"  json:"protocol,omitempty"` //multi protocol support, split by ','
	InterfaceName string           `required:"true"  yaml:"interface"  json:"interface,omitempty"`
	Registries    []ConfigRegistry `required:"true"  yaml:"registries"  json:"registries,omitempty"`
	Cluster       string           `default:"failover" yaml:"cluster"  json:"cluster,omitempty"`
	Loadbalance   string           `default:"random" yaml:"loadbalance"  json:"loadbalance,omitempty"`
	Group         string           `yaml:"group"  json:"group,omitempty"`
	Version       string           `yaml:"version"  json:"version,omitempty"`
	Methods       []struct {
		Name        string `yaml:"name"  json:"name,omitempty"`
		Retries     int64  `yaml:"retries"  json:"retries,omitempty"`
		Loadbalance string `yaml:"loadbalance"  json:"loadbalance,omitempty"`
		Weight      int64  `yaml:"weight"  json:"weight,omitempty"`
	} `yaml:"methods"  json:"methods,omitempty"`
	Warmup        string `yaml:"warmup"  json:"warmup,omitempty"`
	Retries       int64  `yaml:"retries"  json:"retries,omitempty"`
	unexported    *atomic.Bool
	exported      *atomic.Bool
	rpcService    common.RPCService
	exporters     []protocol.Exporter
	cacheProtocol protocol.Protocol
	cacheMutex    sync.Mutex
}

func NewServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		unexported: atomic.NewBool(false),
		exported:   atomic.NewBool(false),
	}

}

func (srvconfig *ServiceConfig) Export() error {
	//TODO: config center start here

	//TODO:delay export
	if srvconfig.unexported != nil && srvconfig.unexported.Load() {
		err := perrors.Errorf("The service %v has already unexported! ", srvconfig.InterfaceName)
		log.Error(err.Error())
		return err
	}
	if srvconfig.unexported != nil && srvconfig.exported.Load() {
		log.Warn("The service %v has already exported! ", srvconfig.InterfaceName)
		return nil
	}

	regUrls := loadRegistries(srvconfig.Registries, providerConfig.Registries, common.PROVIDER)
	urlMap := srvconfig.getUrlMap()

	for _, proto := range loadProtocol(srvconfig.Protocol, providerConfig.Protocols) {
		//registry the service reflect
		methods, err := common.ServiceMap.Register(proto.Name, srvconfig.rpcService)
		if err != nil {
			err := perrors.Errorf("The service %v  export the protocol %v error! Error message is %v .", srvconfig.InterfaceName, proto.Name, err.Error())
			log.Error(err.Error())
			return err
		}
		//contextPath := proto.ContextPath
		//if contextPath == "" {
		//	contextPath = providerConfig.Path
		//}
		url := common.NewURLWithOptions(srvconfig.InterfaceName,
			common.WithProtocol(proto.Name),
			common.WithIp(proto.Ip),
			common.WithPort(proto.Port),
			common.WithParams(urlMap),
			common.WithMethods(strings.Split(methods, ",")))

		for _, regUrl := range regUrls {
			regUrl.SubURL = url

			srvconfig.cacheMutex.Lock()
			if srvconfig.cacheProtocol == nil {
				log.Info("First load the registry protocol!")
				srvconfig.cacheProtocol = extension.GetProtocol("registry")
			}
			srvconfig.cacheMutex.Unlock()

			invoker := extension.GetProxyFactory(providerConfig.ProxyFactory).GetInvoker(*regUrl)
			exporter := srvconfig.cacheProtocol.Export(invoker)
			if exporter == nil {
				panic(perrors.New("New exporter error"))
			}
			srvconfig.exporters = append(srvconfig.exporters, exporter)
		}
	}
	return nil

}

func (srvconfig *ServiceConfig) Implement(s common.RPCService) {
	srvconfig.rpcService = s
}

func (srvconfig *ServiceConfig) getUrlMap() url.Values {
	urlMap := url.Values{}
	urlMap.Set(constant.INTERFACE_KEY, srvconfig.InterfaceName)
	urlMap.Set(constant.TIMESTAMP_KEY, strconv.FormatInt(time.Now().Unix(), 10))
	urlMap.Set(constant.CLUSTER_KEY, srvconfig.Cluster)
	urlMap.Set(constant.LOADBALANCE_KEY, srvconfig.Loadbalance)
	urlMap.Set(constant.WARMUP_KEY, srvconfig.Warmup)
	urlMap.Set(constant.RETRIES_KEY, strconv.FormatInt(srvconfig.Retries, 10))
	urlMap.Set(constant.GROUP_KEY, srvconfig.Group)
	urlMap.Set(constant.VERSION_KEY, srvconfig.Version)
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
		urlMap.Set("methods."+v.Name+"."+constant.RETRIES_KEY, strconv.FormatInt(v.Retries, 10))
		urlMap.Set("methods."+v.Name+"."+constant.WEIGHT_KEY, strconv.FormatInt(v.Weight, 10))
	}

	return urlMap

}
