/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package protocolwrapper

import (
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter/impl"
	"github.com/apache/dubbo-go/protocol"
)

func TestProtocolFilterWrapper_Export(t *testing.T) {
	filtProto := extension.GetProtocol(FILTER)
	filtProto.(*ProtocolFilterWrapper).protocol = &protocol.BaseProtocol{}

	u := common.NewURLWithOptions("Service",
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.SERVICE_FILTER_KEY, impl.ECHO))
	exporter := filtProto.Export(protocol.NewBaseInvoker(*u))
	_, ok := exporter.GetInvoker().(*FilterInvoker)
	assert.True(t, ok)
}

func TestProtocolFilterWrapper_Refer(t *testing.T) {
	filtProto := extension.GetProtocol(FILTER)
	filtProto.(*ProtocolFilterWrapper).protocol = &protocol.BaseProtocol{}

	u := common.NewURLWithOptions("Service",
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.REFERENCE_FILTER_KEY, impl.ECHO))
	invoker := filtProto.Refer(*u)
	_, ok := invoker.(*FilterInvoker)
	assert.True(t, ok)
}
