package dubbo

import (
	"sync"
)

import (
	"github.com/dubbo/dubbo-go/protocol"
)

type DubboExporter struct {
	protocol.BaseExporter
}

func NewDubboExporter(key string, invoker protocol.Invoker, exporterMap *sync.Map) *DubboExporter {
	return &DubboExporter{
		BaseExporter: *protocol.NewBaseExporter(key, invoker, exporterMap),
	}
}
