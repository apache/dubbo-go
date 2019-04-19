package dubbo

import "github.com/dubbo/dubbo-go/protocol"

// wrapping invoker
type DubboExporter struct {
	key     string
	invoker protocol.Invoker
}

func (de *DubboExporter) GetInvoker() protocol.Invoker {
	return de.invoker

}

func (de *DubboExporter) Unexport() {

}
