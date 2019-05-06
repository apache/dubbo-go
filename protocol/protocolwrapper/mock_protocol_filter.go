package protocolwrapper

import (
	"github.com/dubbo/go-for-apache-dubbo/config"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type mockProtocolFilter struct {
}

func NewMockProtocolFilter() protocol.Protocol {
	return &mockProtocolFilter{}
}

func (pfw *mockProtocolFilter) Export(invoker protocol.Invoker) protocol.Exporter {
	return protocol.NewBaseExporter("key", invoker, nil)
}

func (pfw *mockProtocolFilter) Refer(url config.URL) protocol.Invoker {
	return protocol.NewBaseInvoker(url)
}

func (pfw *mockProtocolFilter) Destroy() {

}
