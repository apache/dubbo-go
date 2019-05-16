package protocolwrapper

import (
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/filter/impl"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

func TestProtocolFilterWrapper_Export(t *testing.T) {
	filtProto := extension.GetProtocolExtension(FILTER)
	filtProto.(*ProtocolFilterWrapper).protocol = &protocol.BaseProtocol{}

	u := common.NewURLWithOptions("Service",
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.SERVICE_FILTER_KEY, impl.ECHO))
	exporter := filtProto.Export(protocol.NewBaseInvoker(*u))
	_, ok := exporter.GetInvoker().(*FilterInvoker)
	assert.True(t, ok)
}

func TestProtocolFilterWrapper_Refer(t *testing.T) {
	filtProto := extension.GetProtocolExtension(FILTER)
	filtProto.(*ProtocolFilterWrapper).protocol = &protocol.BaseProtocol{}

	u := common.NewURLWithOptions("Service",
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.REFERENCE_FILTER_KEY, impl.ECHO))
	invoker := filtProto.Refer(*u)
	_, ok := invoker.(*FilterInvoker)
	assert.True(t, ok)
}
