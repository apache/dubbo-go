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

package protocol

import (
	"sync"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common/logger"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	"github.com/dubbo/go-for-apache-dubbo/protocol/protocolwrapper"
	"github.com/dubbo/go-for-apache-dubbo/registry"
	directory2 "github.com/dubbo/go-for-apache-dubbo/registry/directory"
)

var regProtocol *registryProtocol

type registryProtocol struct {
	invokers []protocol.Invoker
	// Registry  Map<RegistryAddress, Registry>
	registries sync.Map
	//To solve the problem of RMI repeated exposure port conflicts, the services that have been exposed are no longer exposed.
	//providerurl <--> exporter
	bounds sync.Map
}

func init() {
	extension.SetProtocol("registry", GetProtocol)
}

func newRegistryProtocol() *registryProtocol {
	return &registryProtocol{
		registries: sync.Map{},
		bounds:     sync.Map{},
	}
}
func getRegistry(regUrl *common.URL) registry.Registry {
	reg, err := extension.GetRegistry(regUrl.Protocol, regUrl)
	if err != nil {
		logger.Errorf("Registry can not connect success, program is going to panic.Error message is %s", err.Error())
		panic(err.Error())
	}
	return reg
}
func (proto *registryProtocol) Refer(url common.URL) protocol.Invoker {

	var registryUrl = url
	var serviceUrl = registryUrl.SubURL
	if registryUrl.Protocol == constant.REGISTRY_PROTOCOL {
		protocol := registryUrl.GetParam(constant.REGISTRY_KEY, "")
		registryUrl.Protocol = protocol
	}
	var reg registry.Registry

	if regI, loaded := proto.registries.Load(registryUrl.Key()); !loaded {
		reg = getRegistry(&registryUrl)
		proto.registries.Store(registryUrl.Key(), reg)
	} else {
		reg = regI.(registry.Registry)
	}

	//new registry directory for store service url from registry
	directory, err := directory2.NewRegistryDirectory(&registryUrl, reg)
	if err != nil {
		logger.Errorf("consumer service %v  create registry directory  error, error message is %s, and will return nil invoker!", serviceUrl.String(), err.Error())
		return nil
	}
	err = reg.Register(*serviceUrl)
	if err != nil {
		logger.Errorf("consumer service %v register registry %v error, error message is %s", serviceUrl.String(), registryUrl.String(), err.Error())
	}
	go directory.Subscribe(*serviceUrl)

	//new cluster invoker
	cluster := extension.GetCluster(serviceUrl.GetParam(constant.CLUSTER_KEY, constant.DEFAULT_CLUSTER))

	invoker := cluster.Join(directory)
	proto.invokers = append(proto.invokers, invoker)
	return invoker
}

func (proto *registryProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	registryUrl := proto.getRegistryUrl(invoker)
	providerUrl := proto.getProviderUrl(invoker)

	var reg registry.Registry

	if regI, loaded := proto.registries.Load(registryUrl.Key()); !loaded {
		reg = getRegistry(&registryUrl)
		proto.registries.Store(registryUrl.Key(), reg)
	} else {
		reg = regI.(registry.Registry)
	}

	err := reg.Register(providerUrl)
	if err != nil {
		logger.Errorf("provider service %v register registry %v error, error message is %s", providerUrl.Key(), registryUrl.Key(), err.Error())
		return nil
	}

	key := providerUrl.Key()
	logger.Infof("The cached exporter keys is %v !", key)
	cachedExporter, loaded := proto.bounds.Load(key)
	if loaded {
		logger.Infof("The exporter has been cached, and will return cached exporter!")
	} else {
		wrappedInvoker := newWrappedInvoker(invoker, providerUrl)
		cachedExporter = extension.GetProtocol(protocolwrapper.FILTER).Export(wrappedInvoker)
		proto.bounds.Store(key, cachedExporter)
		logger.Infof("The exporter has not been cached, and will return a new  exporter!")
	}

	return cachedExporter.(protocol.Exporter)

}

func (proto *registryProtocol) Destroy() {
	for _, ivk := range proto.invokers {
		ivk.Destroy()
	}
	proto.invokers = []protocol.Invoker{}

	proto.bounds.Range(func(key, value interface{}) bool {
		exporter := value.(protocol.Exporter)
		exporter.Unexport()
		proto.bounds.Delete(key)
		return true
	})

	proto.registries.Range(func(key, value interface{}) bool {
		reg := value.(registry.Registry)
		if reg.IsAvailable() {
			reg.Destroy()
		}
		proto.registries.Delete(key)
		return true
	})
}

func (*registryProtocol) getRegistryUrl(invoker protocol.Invoker) common.URL {
	//here add * for return a new url
	url := invoker.GetUrl()
	//if the protocol == registry ,set protocol the registry value in url.params
	if url.Protocol == constant.REGISTRY_PROTOCOL {
		protocol := url.GetParam(constant.REGISTRY_KEY, "")
		url.Protocol = protocol
	}
	return url
}

func (*registryProtocol) getProviderUrl(invoker protocol.Invoker) common.URL {
	url := invoker.GetUrl()
	return *url.SubURL
}

func GetProtocol() protocol.Protocol {
	if regProtocol != nil {
		return regProtocol
	}
	return newRegistryProtocol()
}

type wrappedInvoker struct {
	invoker protocol.Invoker
	url     common.URL
	protocol.BaseInvoker
}

func newWrappedInvoker(invoker protocol.Invoker, url common.URL) *wrappedInvoker {
	return &wrappedInvoker{
		invoker:     invoker,
		url:         url,
		BaseInvoker: *protocol.NewBaseInvoker(common.URL{}),
	}
}
func (ivk *wrappedInvoker) GetUrl() common.URL {
	return ivk.url
}
func (ivk *wrappedInvoker) getInvoker() protocol.Invoker {
	return ivk.invoker
}
