package dubbo

import (
	"context"
)

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/dubbo-go/protocol"
)

// wrapping invoker
type DubboExporter struct {
	ctx         context.Context
	key         string
	invoker     protocol.Invoker
	exporterMap map[string]protocol.Exporter
}

func NewDubboExporter(ctx context.Context, key string, invoker protocol.Invoker, exporterMap map[string]protocol.Exporter) *DubboExporter {
	return &DubboExporter{
		ctx:         ctx,
		key:         key,
		invoker:     invoker,
		exporterMap: exporterMap,
	}
}

func (de *DubboExporter) GetInvoker() protocol.Invoker {
	return de.invoker

}

func (de *DubboExporter) Unexport() {
	log.Info("DubboExporter unexport.")
	de.invoker.Destroy()
	delete(de.exporterMap, de.key)
}
