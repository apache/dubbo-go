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

package failover

import (
	"context"
	"fmt"
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	clusterpkg "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/static"
	"dubbo.apache.org/dubbo-go/v3/cluster/loadbalance/random"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// nolint
func normalInvoke(successCount int, urlParam url.Values, invocations ...*invocation.RPCInvocation) protocol.Result {
	extension.SetLoadbalance("random", random.NewRandomLoadBalance)
	failoverCluster := newFailoverCluster()

	var invokers []protocol.Invoker
	for i := 0; i < 10; i++ {
		newUrl, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i), common.WithParams(urlParam))
		invokers = append(invokers, clusterpkg.NewMockInvoker(newUrl, successCount))
	}

	staticDir := static.NewDirectory(invokers)
	clusterInvoker := failoverCluster.Join(staticDir)
	if len(invocations) > 0 {
		return clusterInvoker.Invoke(context.Background(), invocations[0])
	}
	return clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
}

// nolint
func TestFailoverInvokeSuccess(t *testing.T) {
	urlParams := url.Values{}
	result := normalInvoke(3, urlParams)
	assert.NoError(t, result.Error())
	clusterpkg.Count = 0
}

// nolint
func TestFailoverInvokeFail(t *testing.T) {
	urlParams := url.Values{}
	result := normalInvoke(4, urlParams)
	assert.Errorf(t, result.Error(), "error")
	clusterpkg.Count = 0
}

// nolint
func TestFailoverInvoke1(t *testing.T) {
	urlParams := url.Values{}
	urlParams.Set(constant.RetriesKey, "3")
	result := normalInvoke(4, urlParams)
	assert.NoError(t, result.Error())
	clusterpkg.Count = 0
}

// nolint
func TestFailoverInvoke2(t *testing.T) {
	urlParams := url.Values{}
	urlParams.Set(constant.RetriesKey, "2")
	urlParams.Set("methods.test."+constant.RetriesKey, "3")

	ivc := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))
	result := normalInvoke(4, urlParams, ivc)
	assert.NoError(t, result.Error())
	clusterpkg.Count = 0
}

// nolint
func TestFailoverDestroy(t *testing.T) {
	extension.SetLoadbalance("random", random.NewRandomLoadBalance)
	failoverCluster := newFailoverCluster()

	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		u, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, clusterpkg.NewMockInvoker(u, 1))
	}

	staticDir := static.NewDirectory(invokers)
	clusterInvoker := failoverCluster.Join(staticDir)
	assert.Equal(t, true, clusterInvoker.IsAvailable())
	result := clusterInvoker.Invoke(context.Background(), &invocation.RPCInvocation{})
	assert.NoError(t, result.Error())
	clusterpkg.Count = 0
	clusterInvoker.Destroy()
	assert.Equal(t, false, clusterInvoker.IsAvailable())
}
