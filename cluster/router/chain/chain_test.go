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

package chain

import (
	"fmt"
	_ "github.com/apache/dubbo-go/cluster/router"
	_ "github.com/apache/dubbo-go/cluster/router/tag"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
	"github.com/dubbogo/go-zookeeper/zk"
	"github.com/stretchr/testify/assert"
	"testing"
)

var (
	url, _ = common.NewURL(
		fmt.Sprintf("dubbo://%s:%d/com.ikurento.user.UserProvider", constant.LOCAL_HOST_VALUE, constant.DEFAULT_PORT))
	anyURL, _ = common.NewURL(fmt.Sprintf("condition://%s/com.foo.BarService", constant.ANYHOST_VALUE))
)

const (
	test1234IP = "1.2.3.4"
	test0000IP = "0.0.0.0"
	port20000  = 20000

	dubboForamt      = "dubbo://%s:%d/com.foo.BarService"
	anyUrlFormat     = "condition://%s/com.foo.BarService"
	applicationKey   = "test-condition"
	applicationField = "application"
	forceField       = "force"
	forceValue       = "true"
)

var zkCluster *zk.TestCluster

func TestNewRouterChain(t *testing.T) {
	chain, _ := NewRouterChain(getRouteUrl(applicationKey))
	assert.Equal(t, chain.routerStatus.Load(), int32(HasRouter))
	var invokers []protocol.Invoker
	dubboURL, _ := common.NewURL(fmt.Sprintf(dubboForamt, test1234IP, port20000))
	invokers = append(invokers, protocol.NewBaseInvoker(dubboURL))
	chain.SetInvokers(invokers)
}

func getRouteUrl(applicationKey string) *common.URL {
	url, _ := common.NewURL(fmt.Sprintf(anyUrlFormat, test0000IP))
	url.AddParam(applicationField, applicationKey)
	url.AddParam(forceField, forceValue)
	return url
}
