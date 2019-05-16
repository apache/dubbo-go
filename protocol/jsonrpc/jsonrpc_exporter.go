package jsonrpc

import (
	"sync"
)

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
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

func (je *JsonrpcExporter) Unexport() {
	service := je.GetInvoker().GetUrl().GetParam(constant.INTERFACE_KEY, "")
	je.BaseExporter.Unexport()
	err := common.ServiceMap.UnRegister(JSONRPC, service)
	if err != nil {
		log.Error("[JsonrpcExporter.Unexport] error: %v", err)
	}
}
