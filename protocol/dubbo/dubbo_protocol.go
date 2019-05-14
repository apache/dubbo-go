package dubbo

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

const DUBBO = "dubbo"

func init() {
	extension.SetProtocol(DUBBO, GetProtocol)
}

var dubboProtocol *DubboProtocol

type DubboProtocol struct {
	protocol.BaseProtocol
	serverMap map[string]*Server
}

func NewDubboProtocol() *DubboProtocol {
	return &DubboProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*Server),
	}
}

func (dp *DubboProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetUrl()
	serviceKey := url.Key()
	exporter := NewDubboExporter(serviceKey, invoker, dp.ExporterMap())
	dp.SetExporterMap(serviceKey, exporter)
	log.Info("Export service: %s", url.String())

	// start server
	dp.openServer(url)
	return exporter
}

func (dp *DubboProtocol) Refer(url common.URL) protocol.Invoker {
	invoker := NewDubboInvoker(url, NewClient())
	dp.SetInvokers(invoker)
	log.Info("Refer service: %s", url.String())
	return invoker
}

func (dp *DubboProtocol) Destroy() {
	log.Info("DubboProtocol destroy.")

	dp.BaseProtocol.Destroy()

	// stop server
	for key, server := range dp.serverMap {
		delete(dp.serverMap, key)
		server.Stop()
	}
}

func (dp *DubboProtocol) openServer(url common.URL) {
	exporter, ok := dp.ExporterMap().Load(url.Key())
	if !ok {
		panic("[DubboProtocol]" + url.Key() + "is not existing")
	}
	srv := NewServer(exporter.(protocol.Exporter))
	dp.serverMap[url.Location] = srv
	srv.Start(url)
}

func GetProtocol() protocol.Protocol {
	if dubboProtocol != nil {
		return dubboProtocol
	}
	return NewDubboProtocol()
}
