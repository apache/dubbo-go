package dubbo

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
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
	url := invoker.GetUrl().(*config.URL)
	serviceKey := url.Key()
	exporter := NewDubboExporter(serviceKey, invoker, dp.ExporterMap())
	dp.SetExporterMap(serviceKey, exporter)
	log.Info("Export service: ", url.String())

	// start server
	dp.openServer(*url)
	return exporter
}

func (dp *DubboProtocol) Refer(url config.IURL) protocol.Invoker {
	invoker := NewDubboInvoker(url, NewClient())
	dp.SetInvokers(invoker)
	log.Info("Refer service: ", url.(*config.URL).String())
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

func (dp *DubboProtocol) openServer(url config.URL) {
	srv := NewServer(dp.ExporterMap()[url.Key()])
	dp.serverMap[url.Location] = srv
	srv.Start(url)
}

func GetProtocol() protocol.Protocol {
	if dubboProtocol != nil {
		return dubboProtocol
	}
	return NewDubboProtocol()
}
