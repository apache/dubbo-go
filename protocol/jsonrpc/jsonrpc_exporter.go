package jsonrpc

import (
	"sync"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type JsonrpcExporter struct {
	protocol.BaseExporter
}

func NewJsonrpcExporter(key string, invoker protocol.Invoker, exporterMap *sync.Map) *JsonrpcExporter {
	return &JsonrpcExporter{
		BaseExporter: *protocol.NewBaseExporter(key, invoker, exporterMap),
	}
}
