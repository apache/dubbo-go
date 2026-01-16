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

package static

import (
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestStaticDirList(t *testing.T) {
	invokers := []base.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, base.NewBaseInvoker(url))
	}

	staticDir := NewDirectory(invokers)
	list := staticDir.List(&invocation.RPCInvocation{})

	assert.Len(t, list, 10)
}

func TestStaticDirDestroy(t *testing.T) {
	invokers := []base.Invoker{}
	for i := 0; i < 10; i++ {
		url, _ := common.NewURL(fmt.Sprintf("dubbo://192.168.1.%v:20000/com.ikurento.user.UserProvider", i))
		invokers = append(invokers, base.NewBaseInvoker(url))
	}

	staticDir := NewDirectory(invokers)
	assert.True(t, staticDir.IsAvailable())
	staticDir.Destroy()
	assert.False(t, staticDir.IsAvailable())
}
