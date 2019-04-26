package jsonrpc

import (
	"github.com/dubbo/dubbo-go/protocol"
)

type JsonrpcExporter struct {
	protocol.BaseExporter
}

func NewJsonrpcExporter(key string, invoker protocol.Invoker, exporterMap map[string]protocol.Exporter) *JsonrpcExporter {
	return &JsonrpcExporter{
		BaseExporter: *protocol.NewBaseExporter(key, invoker, exporterMap),
	}
}
