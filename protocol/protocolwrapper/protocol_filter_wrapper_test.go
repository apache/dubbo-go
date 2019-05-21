// Copyright 2016-2019 Yincheng Fang
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
