package dubbo

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

type DubboExporter struct {
	protocol.BaseExporter
}

func NewDubboExporter(key string, invoker protocol.Invoker, exporterMap *sync.Map) *DubboExporter {
	return &DubboExporter{
		BaseExporter: *protocol.NewBaseExporter(key, invoker, exporterMap),
	}
}

func (de *DubboExporter) Unexport() {
	service := de.GetInvoker().GetUrl().GetParam(constant.INTERFACE_KEY, "")
	de.BaseExporter.Unexport()
	err := common.ServiceMap.UnRegister(DUBBO, service)
	if err != nil {
		log.Error("[DubboExporter.Unexport] error: %v", err)
	}
}
