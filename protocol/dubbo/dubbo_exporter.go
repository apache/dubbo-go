package dubbo

import (
	"context"
)

import (
	"github.com/dubbo/dubbo-go/protocol"
)

// wrapping invoker
type DubboExporter struct {
	ctx     context.Context
	key     string
	invoker protocol.Invoker
}

func NewDubboExporter(ctx context.Context, key string, invoker protocol.Invoker) *DubboExporter {
	return &DubboExporter{
		ctx:     ctx,
		key:     key,
		invoker: invoker,
	}
}

func (de *DubboExporter) GetInvoker() protocol.Invoker {
	return de.invoker

}

func (de *DubboExporter) Unexport() {
	de.invoker.Destroy()
}
