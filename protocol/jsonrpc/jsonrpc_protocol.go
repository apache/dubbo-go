package jsonrpc

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/config/support"
	"github.com/dubbo/dubbo-go/protocol"
)

const JSONRPC = "jsonrpc"

func init() {
	extension.SetProtocol(JSONRPC, GetProtocol)
}

var jsonrpcProtocol *JsonrpcProtocol

type JsonrpcProtocol struct {
	protocol.BaseProtocol
	serverMap map[string]*Server
}

func NewDubboProtocol() *JsonrpcProtocol {
	return &JsonrpcProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*Server),
	}
}

func (jp *JsonrpcProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetUrl().(*config.URL)
	serviceKey := url.Key()
	exporter := NewJsonrpcExporter(serviceKey, invoker, jp.ExporterMap())
	jp.SetExporterMap(serviceKey, exporter)
	log.Info("Export service: %s", url.String())

	// start server
	jp.openServer(*url)

	return exporter
}

func (jp *JsonrpcProtocol) Refer(url config.IURL) protocol.Invoker {
	invoker := NewJsonrpcInvoker(url, NewHTTPClient(&HTTPOptions{
		HandshakeTimeout: support.GetConsumerConfig().ConnectTimeout,
		HTTPTimeout:      support.GetConsumerConfig().RequestTimeout,
	}))
	jp.SetInvokers(invoker)
	log.Info("Refer service: %s", url.(*config.URL).String())
	return invoker
}

func (jp *JsonrpcProtocol) Destroy() {
	log.Info("jsonrpcProtocol destroy.")

	jp.BaseProtocol.Destroy()

	// stop server
	for key, server := range jp.serverMap {
		delete(jp.serverMap, key)
		server.Stop()
	}
}

func (jp *JsonrpcProtocol) openServer(url config.URL) {
	exporter, ok := jp.ExporterMap().Load(url.Key())
	if !ok {
		panic("[JsonrpcProtocol]" + url.Key() + "is not existing")
	}
	srv := NewServer(exporter.(protocol.Exporter))
	jp.serverMap[url.Location] = srv
	srv.Start(url)
}

func GetProtocol() protocol.Protocol {
	if jsonrpcProtocol != nil {
		return jsonrpcProtocol
	}
	return NewDubboProtocol()
}
