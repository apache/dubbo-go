package protocol

import (
	"sync"
)

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/dubbo-go/common/constant"
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
	"github.com/dubbo/dubbo-go/protocol/protocolwrapper"
	"github.com/dubbo/dubbo-go/registry"
	directory2 "github.com/dubbo/dubbo-go/registry/directory"
)

var registryProtocol *RegistryProtocol

type RegistryProtocol struct {
	// Registry  Map<RegistryAddress, Registry>
	registries sync.Map
	//To solve the problem of RMI repeated exposure port conflicts, the services that have been exposed are no longer exposed.
	//providerurl <--> exporter
	bounds sync.Map
}

func init() {
	extension.SetProtocol("registry", GetProtocol)
}

func NewRegistryProtocol() *RegistryProtocol {
	return &RegistryProtocol{
		registries: sync.Map{},
		bounds:     sync.Map{},
	}
}
func getRegistry(regUrl *config.URL) registry.Registry {
	reg, err := extension.GetRegistryExtension(regUrl.Protocol, regUrl)
	if err != nil {
		log.Error("Registry can not connect success, program is going to panic.Error message is %s", err.Error())
		panic(err.Error())
	}
	return reg
}
func (proto *RegistryProtocol) Refer(url config.URL) protocol.Invoker {

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
		log.Error("consumer service %v  create registry directory  error, error message is %s, and will return nil invoker!", serviceUrl.String(), err.Error())
		return nil
	}
	err = reg.Register(*serviceUrl)
	if err != nil {
		log.Error("consumer service %v register registry %v error, error message is %s", serviceUrl.String(), registryUrl.String(), err.Error())
	}
	go directory.Subscribe(*serviceUrl)

	//new cluster invoker
	cluster := extension.GetCluster(serviceUrl.Params.Get(constant.CLUSTER_KEY))

	return cluster.Join(directory)
}

func (proto *RegistryProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
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
		log.Error("provider service %v register registry %v error, error message is %s", providerUrl.Key(), registryUrl.Key(), err.Error())
		return nil
	}

	key := providerUrl.Key()
	log.Info("The cached exporter keys is %v !", key)
	cachedExporter, loaded := proto.bounds.Load(key)
	if loaded {
		log.Info("The exporter has been cached, and will return cached exporter!")
	} else {
		wrappedInvoker := newWrappedInvoker(invoker, providerUrl)
		cachedExporter = extension.GetProtocolExtension(protocolwrapper.FILTER).Export(wrappedInvoker)
		proto.bounds.Store(key, cachedExporter)
		log.Info("The exporter has not been cached, and will return a new  exporter!")
	}

	return cachedExporter.(protocol.Exporter)

}

func (*RegistryProtocol) Destroy() {

}

func (*RegistryProtocol) getRegistryUrl(invoker protocol.Invoker) config.URL {
	//here add * for return a new url
	url := invoker.GetUrl()
	//if the protocol == registry ,set protocol the registry value in url.params
	if url.Protocol == constant.REGISTRY_PROTOCOL {
		protocol := url.GetParam(constant.REGISTRY_KEY, "")
		url.Protocol = protocol
	}
	return url
}

func (*RegistryProtocol) getProviderUrl(invoker protocol.Invoker) config.URL {
	url := invoker.GetUrl()
	return *url.SubURL
}

func GetProtocol() protocol.Protocol {
	if registryProtocol != nil {
		return registryProtocol
	}
	return NewRegistryProtocol()
}

type wrappedInvoker struct {
	invoker protocol.Invoker
	url     config.URL
	protocol.BaseInvoker
}

func newWrappedInvoker(invoker protocol.Invoker, url config.URL) *wrappedInvoker {
	return &wrappedInvoker{
		invoker:     invoker,
		url:         url,
		BaseInvoker: *protocol.NewBaseInvoker(config.URL{}),
	}
}
func (ivk *wrappedInvoker) GetUrl() config.URL {
	return ivk.url
}
func (ivk *wrappedInvoker) getInvoker() protocol.Invoker {
	return ivk.invoker
}
