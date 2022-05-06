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
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestConusmerFilterInvoke(t *testing.T) {
	url := common.NewURLWithOptions(common.WithParams(url.Values{}))
	invocation := invocation.NewRPCInvocation("GetUser", []interface{}{"OK"}, make(map[string]interface{}))

	rootConfig := config.NewRootConfigBuilder().
		SetShutDown(config.NewShutDownConfigBuilder().
			SetTimeout("60s").
			SetStepTimeout("3s").
			Build()).Build()

	config.SetRootConfig(*rootConfig)

	filterValue, _ := extension.GetFilter(constant.GracefulShutdownConsumerFilterKey)
	filter := filterValue.(*consumerGracefulShutdownFilter)
	filter.Set(constant.GracefulShutdownFilterShutdownConfig, config.GetShutDown())
	assert.Equal(t, filter.shutdownConfig, config.GetShutDown())

	result := filter.Invoke(context.Background(), protocol.NewBaseInvoker(url), invocation)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())
}
