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

package filter_impl

import (
	"context"
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/filter"
	common2 "dubbo.apache.org/dubbo-go/v3/filter/handler"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestGenericFilterInvoke(t *testing.T) {
	invoc := invocation.NewRPCInvocation("GetUser", []interface{}{"OK"}, make(map[string]interface{}))
	invokeUrl := common.NewURLWithOptions(common.WithParams(url.Values{}))

	shutdownFilter := extension.GetFilter(constant.PROVIDER_SHUTDOWN_FILTER).(*gracefulShutdownFilter)

	providerConfig := config.GetProviderConfig()

	assert.False(t, shutdownFilter.rejectNewRequest())
	assert.Nil(t, providerConfig.ShutdownConfig)

	assert.Equal(t, extension.GetRejectedExecutionHandler(constant.DEFAULT_KEY),
		shutdownFilter.getRejectHandler())

	result := shutdownFilter.Invoke(context.Background(), protocol.NewBaseInvoker(invokeUrl), invoc)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())

	providerConfig.ShutdownConfig = &config.ShutdownConfig{
		RejectRequest:        true,
		RejectRequestHandler: "mock",
	}
	shutdownFilter.shutdownConfig = providerConfig.ShutdownConfig

	assert.True(t, shutdownFilter.rejectNewRequest())
	result = shutdownFilter.OnResponse(context.Background(), nil, protocol.NewBaseInvoker(invokeUrl), invoc)
	assert.Nil(t, result)

	rejectHandler := &common2.OnlyLogRejectedExecutionHandler{}
	extension.SetRejectedExecutionHandler("mock", func() filter.RejectedExecutionHandler {
		return rejectHandler
	})
	assert.True(t, providerConfig.ShutdownConfig.RequestsFinished)
	assert.Equal(t, rejectHandler, shutdownFilter.getRejectHandler())
}
