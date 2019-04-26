package dubbo

import (
	"context"
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
	ctx         context.Context
	exporterMap map[string]protocol.Exporter
	invokers    []protocol.Invoker
}

func NewDubboProtocol(ctx context.Context) *DubboProtocol {
	return &DubboProtocol{ctx: ctx, exporterMap: make(map[string]protocol.Exporter)}
}

func (dp *DubboProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetUrl().(*config.URL)
	serviceKey := url.Key()
	exporter := NewDubboExporter(nil, serviceKey, invoker, dp.exporterMap)
	dp.exporterMap[serviceKey] = exporter
	log.Info("Export service: ", url.String())

	// start server
	dp.openServer(*url)
	return exporter
}

func (dp *DubboProtocol) Refer(url config.URL) protocol.Invoker {
	invoker := NewDubboInvoker(nil, url, getClient())
	dp.invokers = append(dp.invokers, invoker)
	log.Info("Refer service: ", url.String())
	return invoker
}

func (dp *DubboProtocol) Destroy() {
	log.Info("DubboProtocol destroy.")
	srv.Stop() // stop server
}

func (dp *DubboProtocol) openServer(url config.URL) {
	srv.Start(url)
}

func GetProtocol() protocol.Protocol {
	if dubboProtocol != nil {
		return dubboProtocol
	}
	return NewDubboProtocol(nil)
}

func getClient() *Client {
	return NewClient()
}
