package registry

import (
	"sync"
)

import (
	log "github.com/AlexStocks/log4go"
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
	extension.SetProtocol("registry", NewRegistryProtocol)
}

func NewRegistryProtocol() protocol.Protocol {
	return &RegistryProtocol{
		registies: make(map[string]Registry),
	}
}

func (protocol *RegistryProtocol) Refer(url config.IURL) protocol.Invoker {
	var regUrl = url.(*config.RegistryURL)
	var serviceUrl = regUrl.URL

	protocol.registiesMutex.Lock()
	defer protocol.registiesMutex.Unlock()
	var reg Registry
	var ok bool

	if reg, ok = protocol.registies[url.Key()]; !ok {
		var err error
		reg, err = extension.GetRegistryExtension(regUrl.Protocol, regUrl)
		if err != nil {
			log.Error("Registry can not connect success, program is going to panic.Error message is %s", err.Error())
			panic(err.Error())
		} else {
			protocol.registies[url.Key()] = reg
		}
	}
	//new registry directory for store service url from registry
	directory := NewRegistryDirectory(regUrl, reg)
	go directory.subscribe(serviceUrl)

	//new cluster invoker
	cluster := extension.GetCluster(serviceUrl.Cluster)
	return cluster.Join(directory)
}

func (*RegistryProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	return nil
}

func (*RegistryProtocol) Destroy() {
}
