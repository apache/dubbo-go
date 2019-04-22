package registry

import (
	"github.com/juju/utils/registry"
	"github.com/prometheus/common/log"
	"sync"
	"time"
)

import (
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
)

const RegistryConnDelay = 3

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

func (protocol *RegistryProtocol) Refer(url config.IURL) (Registry, error) {
	var regUrl = url.(*config.RegistryURL)

	protocol.registiesMutex.Lock()
	var reg Registry
	if reg, ok := protocol.registies[url.Key()]; !ok {

		var err error
		reg, err = extension.GetRegistryExtension(regUrl.Protocol, regUrl)
		protocol.registies[url.Key()] = reg
		if err != nil {
			return nil, err
		}
	}
	protocol.subscribe(reg, regUrl.URL)
}

func (*RegistryProtocol) Export() {

}

func (*RegistryProtocol) Destroy() {
}
