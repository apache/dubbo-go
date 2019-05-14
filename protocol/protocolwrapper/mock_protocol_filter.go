package protocolwrapper

import (
	"sync"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type mockProtocolFilter struct {
}

func NewMockProtocolFilter() protocol.Protocol {
	return &mockProtocolFilter{}
}

func (pfw *mockProtocolFilter) Export(invoker protocol.Invoker) protocol.Exporter {
	return protocol.NewBaseExporter("key", invoker, &sync.Map{})
}

func (pfw *mockProtocolFilter) Refer(url common.URL) protocol.Invoker {
	return protocol.NewBaseInvoker(url)
}

func (pfw *mockProtocolFilter) Destroy() {

}
