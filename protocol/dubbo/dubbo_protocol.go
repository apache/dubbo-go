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
	exporterMap map[string]protocol.Exporter
	invokers    []protocol.Invoker
}

func NewDubboProtocol() protocol.Protocol {
	return &DubboProtocol{exporterMap: make(map[string]protocol.Exporter)}
}

func (dp *DubboProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetURL()
	serviceKey := url.Key()
	exporter := &DubboExporter{invoker: invoker, key: serviceKey}
	dp.exporterMap[serviceKey] = exporter
	log.Info("Export service: ", url.String())

	// start server
	dp.openServer(url)
	return exporter
}

func (dp *DubboProtocol) Refer(url config.ConfigURL) protocol.Invoker {
	invoker := &DubboInvoker{url: url}
	dp.invokers = append(dp.invokers, invoker)
	log.Info("Refer service: ", url.String())
	return invoker
}

func (dp *DubboProtocol) Destroy() {

}

func (dp *DubboProtocol) openServer(url config.ConfigURL) {
	srv.Start(url)
}

func GetProtocol() protocol.Protocol {
	if dubboProtocol != nil {
		return dubboProtocol
	}
	return NewDubboProtocol()
}
