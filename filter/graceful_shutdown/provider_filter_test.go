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

package graceful_shutdown

import (
	"context"
	"net/url"
	"testing"
)

import (
	perrors "github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

func TestProviderFilterInvokeWithGlobalPackage(t *testing.T) {
	var (
		url            = common.NewURLWithOptions(common.WithParams(url.Values{}))
		invocation     = invocation.NewRPCInvocation("GetUser", []any{"OK"}, make(map[string]any))
		opt            = graceful_shutdown.NewOptions()
		filterValue, _ = extension.GetFilter(constant.GracefulShutdownProviderFilterKey)
	)

	extension.SetRejectedExecutionHandler("test", func() filter.RejectedExecutionHandler {
		return &TestRejectedExecutionHandler{}
	})

	opt.Shutdown.RejectRequestHandler = "test"

	filter := filterValue.(*providerGracefulShutdownFilter)
	filter.Set(constant.GracefulShutdownFilterShutdownConfig, opt.Shutdown)
	assert.Equal(t, filter.shutdownConfig, opt.Shutdown)

	result := filter.Invoke(context.Background(), base.NewBaseInvoker(url), invocation)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())

	opt.Shutdown.RejectRequest.Store(true)
	result = filter.Invoke(context.Background(), base.NewBaseInvoker(url), invocation)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Error().Error(), "Rejected")
}

// only for compatibility with old config, able to directly remove after config is deleted
func TestProviderFilterInvokeWithConfigPackage(t *testing.T) {
	var (
		url        = common.NewURLWithOptions(common.WithParams(url.Values{}))
		invocation = invocation.NewRPCInvocation("GetUser", []any{"OK"}, make(map[string]any))
		rootConfig = config.NewRootConfigBuilder().
				SetShutDown(config.NewShutDownConfigBuilder().
					SetTimeout("60s").
					SetStepTimeout("3s").
					SetRejectRequestHandler("test").
					Build()).Build()
		filterValue, _ = extension.GetFilter(constant.GracefulShutdownProviderFilterKey)
	)

	extension.SetRejectedExecutionHandler("test", func() filter.RejectedExecutionHandler {
		return &TestRejectedExecutionHandler{}
	})

	config.SetRootConfig(*rootConfig)

	filter := filterValue.(*providerGracefulShutdownFilter)
	filter.Set(constant.GracefulShutdownFilterShutdownConfig, config.GetShutDown())
	assert.Equal(t, filter.shutdownConfig, compatGlobalShutdownConfig(config.GetShutDown()))

	result := filter.Invoke(context.Background(), base.NewBaseInvoker(url), invocation)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())

	// only use this way to set the RejectRequest, because the variable is compact to GlobalShutdownConfig
	filter.shutdownConfig.RejectRequest.Store(true)
	// not able to use this way to set the RejectRequest
	//config.GetShutDown().RejectRequest.Store(true)

	result = filter.Invoke(context.Background(), base.NewBaseInvoker(url), invocation)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Error().Error(), "Rejected")
}

type TestRejectedExecutionHandler struct{}

// RejectedExecution will do nothing, it only log the invocation.
func (handler *TestRejectedExecutionHandler) RejectedExecution(url *common.URL,
	_ base.Invocation) result.Result {

	return &result.RPCResult{
		Err: perrors.New("Rejected"),
	}
}
