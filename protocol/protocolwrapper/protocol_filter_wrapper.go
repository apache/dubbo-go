package protocolwrapper

import (
	"strings"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/filter"
	"github.com/dubbo/go-for-apache-dubbo/filter/impl"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
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

func (pfw *ProtocolFilterWrapper) Refer(url common.URL) protocol.Invoker {
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
	if filtName != "" && strings.LastIndex(filtName, ",") != len(filtName)-1 {
		filtName = filtName + ","
	}
	if key == constant.SERVICE_FILTER_KEY { // echofilter must be the first in provider
		filtName = impl.ECHO + "," + filtName
	}
	filtNames := strings.Split(filtName, ",")
	next := invoker
	// The order of filters is from left to right, so loading from right to left
	for i := len(filtNames) - 2; i >= 0; i-- {
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

func (fi *FilterInvoker) GetUrl() common.URL {
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
