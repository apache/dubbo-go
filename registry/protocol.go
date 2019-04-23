package registry

import (
	"context"
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
	context context.Context
	// Registry  Map<RegistryAddress, Registry>
	registies      map[string]Registry
	registiesMutex sync.Mutex
}

func init() {
	extension.SetRefProtocol(NewRegistryProtocol)
}

func NewRegistryProtocol(ctx context.Context) protocol.Protocol {
	return &RegistryProtocol{
		context:   ctx,
		registies: make(map[string]Registry),
	}
}

func (protocol *RegistryProtocol) Refer(url config.IURL) (protocol.Invoker, error) {
	var regUrl = url.(*config.RegistryURL)
	var serviceUrl = regUrl.URL

	protocol.registiesMutex.Lock()
	defer protocol.registiesMutex.Unlock()
	var reg Registry
	var ok bool

	if reg, ok = protocol.registies[url.Key()]; !ok {
		var err error
		reg, err = extension.GetRegistryExtension(regUrl.Protocol, protocol.context, regUrl)
		if err != nil {
			return nil, err
		} else {
			protocol.registies[url.Key()] = reg
		}
	}
	//new registry directory for store service url from registry
	directory := NewRegistryDirectory(protocol.context, regUrl, reg)
	go directory.subscribe(serviceUrl)

	//new cluster invoker
	cluster := extension.GetCluster(serviceUrl.Cluster, protocol.context)
	return cluster.Join(directory), nil
}

func (*RegistryProtocol) Export() {

}

func (*RegistryProtocol) Destroy() {
}
