// Copyright 2016-2019 hxmhlt
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
package proxy_factory

import (
	"testing"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	"github.com/stretchr/testify/assert"
)

func Test_GetProxy(t *testing.T) {
	proxyFactory := NewDefaultProxyFactory()
	url := common.NewURLWithOptions("testservice")
	proxy := proxyFactory.GetProxy(protocol.NewBaseInvoker(*url), url)
	assert.NotNil(t, proxy)
}

func Test_GetInvoker(t *testing.T) {
	proxyFactory := NewDefaultProxyFactory()
	url := common.NewURLWithOptions("testservice")
	invoker := proxyFactory.GetInvoker(*url)
	assert.True(t, invoker.IsAvailable())
}
