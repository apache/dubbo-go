package protocol

import (
	"github.com/dubbo/dubbo-go/protocol/protocolwrapper"
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
	"github.com/dubbo/dubbo-go/registry"
	directory2 "github.com/dubbo/dubbo-go/registry/directory"
)

const RegistryConnDelay = 3

var registryProtocol *RegistryProtocol

type RegistryProtocol struct {
	// Registry  Map<RegistryAddress, Registry>
	registies sync.Map
}

func init() {
	extension.SetProtocol("registry", GetProtocol)
}

func NewRegistryProtocol() *RegistryProtocol {
	return &RegistryProtocol{
		registies: sync.Map{},
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
func (protocol *RegistryProtocol) Refer(url config.URL) protocol.Invoker {
	var regUrl = url
	var serviceUrl = regUrl.SubURL

	var reg registry.Registry

	regI, _ := protocol.registies.LoadOrStore(url.Key(),
		getRegistry(&regUrl))
	reg = regI.(registry.Registry)

	//new registry directory for store service url from registry
	directory := directory2.NewRegistryDirectory(&regUrl, reg)
	go directory.Subscribe(*serviceUrl)

	//new cluster invoker
	cluster := extension.GetCluster(serviceUrl.Params.Get(constant.CLUSTER_KEY))
	return cluster.Join(directory)
}

func (protocol *RegistryProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	registryUrl := protocol.getRegistryUrl(invoker)
	providerUrl := protocol.getProviderUrl(invoker)

	regI, _ := protocol.registies.LoadOrStore(providerUrl.Key(),
		getRegistry(&registryUrl))

	reg := regI.(registry.Registry)

	err := reg.Register(providerUrl)
	if err != nil {
		log.Error("provider service %v register registry %v error, error message is %v", providerUrl.String(), registryUrl.String(), err.Error())
	}

	wrappedInvoker := newWrappedInvoker(invoker, providerUrl)

	return extension.GetProtocolExtension(protocolwrapper.FILTER).Export(wrappedInvoker)

}

func (*RegistryProtocol) Destroy() {
}

func (*RegistryProtocol) getRegistryUrl(invoker protocol.Invoker) config.URL {
	//here add * for return a new url
	url := invoker.GetUrl()
	//if the protocol == registry ,set protocol the registry value in url.params
	if url.Protocol == constant.REGISTRY_PROTOCOL {
		protocol := url.GetParam(constant.REGISTRY_KEY, constant.DEFAULT_PROTOCOL)
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
		BaseInvoker: *protocol.NewBaseInvoker(nil),
	}
}
func (ivk *wrappedInvoker) GetUrl() config.URL {
	return ivk.url
}
func (ivk *wrappedInvoker) getInvoker() protocol.Invoker {
	return ivk.invoker
}
