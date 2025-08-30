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
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/graceful_shutdown"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestConsumerFilterInvokeWithGlobalPackage(t *testing.T) {
	var (
		baseUrl        = common.NewURLWithOptions(common.WithParams(url.Values{}))
		rpcInvocation  = invocation.NewRPCInvocation("GetUser", []any{"OK"}, make(map[string]any))
		opt            = graceful_shutdown.NewOptions()
		filterValue, _ = extension.GetFilter(constant.GracefulShutdownConsumerFilterKey)
	)

	opt.Shutdown.RejectRequestHandler = "test"

	filter := filterValue.(*consumerGracefulShutdownFilter)
	filter.Set(constant.GracefulShutdownFilterShutdownConfig, opt.Shutdown)
	assert.Equal(t, filter.shutdownConfig, opt.Shutdown)

	result := filter.Invoke(context.Background(), base.NewBaseInvoker(baseUrl), rpcInvocation)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())
}

// only for compatibility with old config, able to directly remove after config is deleted
func TestConsumerFilterInvokeWithConfigPackage(t *testing.T) {
	var (
		baseUrl       = common.NewURLWithOptions(common.WithParams(url.Values{}))
		rpcInvocation = invocation.NewRPCInvocation("GetUser", []any{"OK"}, make(map[string]any))
		rootConfig    = config.NewRootConfigBuilder().
				SetShutDown(config.NewShutDownConfigBuilder().
					SetTimeout("60s").
					SetStepTimeout("3s").
					Build()).Build()
		filterValue, _ = extension.GetFilter(constant.GracefulShutdownConsumerFilterKey)
	)

	config.SetRootConfig(*rootConfig)

	filter := filterValue.(*consumerGracefulShutdownFilter)
	filter.Set(constant.GracefulShutdownFilterShutdownConfig, config.GetShutDown())
	assert.Equal(t, filter.shutdownConfig, compatGlobalShutdownConfig(config.GetShutDown()))

	result := filter.Invoke(context.Background(), base.NewBaseInvoker(baseUrl), rpcInvocation)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())
}
