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

package cluster_impl

import (
	"context"
	"fmt"
	"net/url"
	"testing"
)
import (
	perrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/cluster/directory"
	"github.com/apache/dubbo-go/cluster/loadbalance"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

/////////////////////////////
// mock invoker
/////////////////////////////

type MockInvoker struct {
	url       common.URL
	available bool
	destroyed bool

	successCount int
}

func NewMockInvoker(url common.URL, successCount int) *MockInvoker {
	return &MockInvoker{
		url:          url,
		available:    true,
		destroyed:    false,
		successCount: successCount,
	}
}

func (bi *MockInvoker) GetUrl() common.URL {
	return bi.url
}

func (bi *MockInvoker) IsAvailable() bool {
	return bi.available
}

func (bi *MockInvoker) IsDestroyed() bool {
	return bi.destroyed
}

type rest struct {
	tried   int
	success bool
}

func (bi *MockInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	count++
	var success bool
	var err error = nil
	if count >= bi.successCount {
		success = true
	} else {
		err = perrors.New("error")
	}
	result := &protocol.RPCResult{Err: err, Rest: rest{tried: count, success: success}}

	return result
}

func (bi *MockInvoker) Destroy() {
	logger.Infof("Destroy invoker: %v", bi.GetUrl().String())
	bi.destroyed = true
	bi.available = false
}

var count int

func normalInvoke(t *testing.T, successCount int, urlParam url.Values, invocations ...*invocation.RPCInvocation) protocol.Result {
	extension.SetLoadbalance("random", loadbalance.NewRandomLoadBalance)
	failoverCluster := NewFailoverCluster()

	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i), common.WithParams(urlParam))
		invokers = append(invokers, NewMockInvoker(url, successCount))
	}

	staticDir := directory.NewStaticDirectory(invokers)
	clusterInvoker := failoverCluster.Join(staticDir)
	if len(invocations) > 0 {
		return clusterInvoker.Invoke(invocations[0])
	}
	return clusterInvoker.Invoke(&invocation.RPCInvocation{})
}
func Test_FailoverInvokeSuccess(t *testing.T) {
	urlParams := url.Values{}
	result := normalInvoke(t, 3, urlParams)
	assert.NoError(t, result.Error())
	count = 0
}

func Test_FailoverInvokeFail(t *testing.T) {
	urlParams := url.Values{}
	result := normalInvoke(t, 4, urlParams)
	assert.Errorf(t, result.Error(), "error")
	count = 0
}

func Test_FailoverInvoke1(t *testing.T) {
	urlParams := url.Values{}
	urlParams.Set(constant.RETRIES_KEY, "3")
	result := normalInvoke(t, 4, urlParams)
	assert.NoError(t, result.Error())
	count = 0
}

func Test_FailoverInvoke2(t *testing.T) {
	urlParams := url.Values{}
	urlParams.Set(constant.RETRIES_KEY, "2")
	urlParams.Set("methods.test."+constant.RETRIES_KEY, "3")

	ivc := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("test"))
	result := normalInvoke(t, 4, urlParams, ivc)
	assert.NoError(t, result.Error())
	count = 0
}

func Test_FailoverDestroy(t *testing.T) {
	extension.SetLoadbalance("random", loadbalance.NewRandomLoadBalance)
	failoverCluster := NewFailoverCluster()

	invokers := []protocol.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(context.TODO(), fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, NewMockInvoker(url, 1))
	}

	staticDir := directory.NewStaticDirectory(invokers)
	clusterInvoker := failoverCluster.Join(staticDir)
	assert.Equal(t, true, clusterInvoker.IsAvailable())
	result := clusterInvoker.Invoke(&invocation.RPCInvocation{})
	assert.NoError(t, result.Error())
	count = 0
	clusterInvoker.Destroy()
	assert.Equal(t, false, clusterInvoker.IsAvailable())

}
