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

package proxy_factory

import (
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

func TestPassThroughProxyFactoryGetProxy(t *testing.T) {
	proxyFactory := NewPassThroughProxyFactory()
	url := common.NewURLWithOptions()
	proxy := proxyFactory.GetProxy(protocol.NewBaseInvoker(url), url)
	assert.NotNil(t, proxy)
}

type TestPassThroughProxyFactoryAsync struct {
}

func (u *TestPassThroughProxyFactoryAsync) CallBack(res common.CallbackResponse) {
	fmt.Println("CallBack res:", res)
}

func TestPassThroughProxyFactoryGetAsyncProxy(t *testing.T) {
	proxyFactory := NewPassThroughProxyFactory()
	url := common.NewURLWithOptions()
	async := &TestPassThroughProxyFactoryAsync{}
	proxy := proxyFactory.GetAsyncProxy(protocol.NewBaseInvoker(url), async.CallBack, url)
	assert.NotNil(t, proxy)
}

func TestPassThroughProxyFactoryGetInvoker(t *testing.T) {
	proxyFactory := NewPassThroughProxyFactory()
	url := common.NewURLWithOptions()
	invoker := proxyFactory.GetInvoker(url)
	assert.True(t, invoker.IsAvailable())
}
