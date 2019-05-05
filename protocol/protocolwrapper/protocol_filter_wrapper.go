package protocolwrapper

import (
	"github.com/dubbo/dubbo-go/filter/imp"
	"strings"
)

import (
	"github.com/dubbo/dubbo-go/common/constant"
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/filter"
	"github.com/dubbo/dubbo-go/protocol"
)

const FILTER = "filter"

func init() {
	extension.SetProtocol(FILTER, GetProtocol)
}

// protocol in url decide who ProtocolFilterWrapper.protocol is
type ProtocolFilterWrapper struct {
	protocol protocol.Protocol
}

func (pfw *ProtocolFilterWrapper) Export(invoker protocol.Invoker) protocol.Exporter {
	if pfw.protocol == nil {
		pfw.protocol = extension.GetProtocolExtension(invoker.GetUrl().Protocol)
	}
	invoker = buildInvokerChain(invoker, constant.SERVICE_FILTER_KEY)
	return pfw.protocol.Export(invoker)
}

func (pfw *ProtocolFilterWrapper) Refer(url config.URL) protocol.Invoker {
	if pfw.protocol == nil {
		pfw.protocol = extension.GetProtocolExtension(url.Protocol)
	}
	return buildInvokerChain(pfw.protocol.Refer(url), constant.REFERENCE_FILTER_KEY)
}

func (pfw *ProtocolFilterWrapper) Destroy() {
	pfw.protocol.Destroy()
}

func buildInvokerChain(invoker protocol.Invoker, key string) protocol.Invoker {
	filtName := invoker.GetUrl().Params.Get(key)
	if filtName == "" { // echo must be the first
		filtName = imp.ECHO
	} else {
		filtName = imp.ECHO + "," + filtName
	}
	filtNames := strings.Split(filtName, ",")
	next := invoker
	// The order of filters is from left to right, so loading from right to left
	for i := len(filtNames) - 1; i >= 0; i-- {
		filter := extension.GetFilterExtension(filtNames[i])
		fi := &FilterInvoker{next: next, invoker: invoker, filter: filter}
		next = fi
	}

	return next
}

func GetProtocol() protocol.Protocol {
	return &ProtocolFilterWrapper{}
}

///////////////////////////
// filter invoker
///////////////////////////

type FilterInvoker struct {
	next    protocol.Invoker
	invoker protocol.Invoker
	filter  filter.Filter
}

func (fi *FilterInvoker) GetUrl() config.URL {
	return fi.invoker.GetUrl()
}

func (fi *FilterInvoker) IsAvailable() bool {
	return fi.invoker.IsAvailable()
}

func (fi *FilterInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	result := fi.filter.Invoke(fi.next, invocation)
	return fi.filter.OnResponse(result, fi.invoker, invocation)
}

func (fi *FilterInvoker) Destroy() {
	fi.invoker.Destroy()
}
