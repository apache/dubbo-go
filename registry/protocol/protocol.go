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
	//registies sync.Map
	//To solve the problem of RMI repeated exposure port conflicts, the services that have been exposed are no longer exposed.
	//providerurl <--> exporter
	bounds sync.Map
}

func init() {
	extension.SetProtocol("registry", GetProtocol)
}

func NewRegistryProtocol() *RegistryProtocol {
	return &RegistryProtocol{
		//registies: sync.Map{},
		bounds: sync.Map{},
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
	var regUrl = url
	var serviceUrl = regUrl.SubURL

	if regUrl.Protocol == constant.REGISTRY_PROTOCOL {
		protocol := regUrl.GetParam(constant.REGISTRY_KEY, "")
		regUrl.Protocol = protocol
	}
	reg := getRegistry(&regUrl)

	//new registry directory for store service url from registry
	directory := directory2.NewRegistryDirectory(&regUrl, reg)
	go directory.Subscribe(*serviceUrl)

	//new cluster invoker
	cluster := extension.GetCluster(serviceUrl.Params.Get(constant.CLUSTER_KEY))
	return cluster.Join(directory)
}

func (proto *RegistryProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	registryUrl := proto.getRegistryUrl(invoker)
	providerUrl := proto.getProviderUrl(invoker)

	reg := getRegistry(&registryUrl)

	err := reg.Register(providerUrl)
	if err != nil {
		log.Error("provider service %v register registry %v error, error message is %s", providerUrl.String(), registryUrl.String(), err.Error())
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
