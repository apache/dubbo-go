package registry

import (
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
	"sync"
)

type RegistryProtocol struct {
	// Registry  Map<RegistryAddress, Registry>
	registies      map[string]Registry
	registiesMutex sync.Mutex
}

func init() {
	extension.SetRefProtocol(NewRegistryProtocol)
}

func NewRegistryProtocol() protocol.Protocol {
	return &RegistryProtocol{
		registies: make(map[string]Registry),
	}
}

func (protocol *RegistryProtocol) Refer(url config.ConfigURL) Registry {
	protocol.registiesMutex.Lock()
	if registry, ok := protocol.registies[url.Key()]; ok {

	} else {
		extension.GetRegistryExtension(url.Protocol(), WithDubboType(CONSUMER), WithApplicationConf())
	}
}

func (*RegistryProtocol) Export() {

}

func (*RegistryProtocol) Destroy() {
}
