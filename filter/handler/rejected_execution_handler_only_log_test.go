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

package handler

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

// TestOnlyLogRejectedExecutionHandler_RejectedExecution checks that a rejected
// invocation fails with Unavailable, and that the error sent to the consumer
// names the service and method without the URL.
func TestOnlyLogRejectedExecutionHandler_RejectedExecution(t *testing.T) {
	handler := GetOnlyLogRejectedExecutionHandler()
	invokeURL := common.NewURLWithOptions(
		common.WithUsername("user"),
		common.WithPassword("secret"),
		common.WithParamsValue(constant.InterfaceKey, "greet.GreetService"),
		common.WithParamsValue("token", "token-value"),
	)

	res := handler.RejectedExecution(invokeURL, invocation.NewRPCInvocation("Greet", nil, nil))
	require.Error(t, res.Error())
	assert.Equal(t, tri.CodeUnavailable, tri.CodeOf(res.Error()))
	msg := res.Error().Error()
	assert.Contains(t, msg, "service=greet.GreetService")
	assert.Contains(t, msg, "method=Greet")
	assert.NotContains(t, msg, "secret")
	assert.NotContains(t, msg, "token-value")
	assert.Nil(t, res.Result())

	t.Run("nil invocation", func(t *testing.T) {
		res := handler.RejectedExecution(invokeURL, nil)
		require.Error(t, res.Error())
		assert.Equal(t, tri.CodeUnavailable, tri.CodeOf(res.Error()))
	})
}
