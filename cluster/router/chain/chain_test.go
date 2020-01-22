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
	"context"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/config"
	"github.com/apache/dubbo-go/common/extension"
	_ "github.com/apache/dubbo-go/config_center/zookeeper"
	"github.com/apache/dubbo-go/remoting/zookeeper"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
	"time"
)

import (
	_ "github.com/apache/dubbo-go/cluster/router/condition"
)

func TestNewRouterChain(t *testing.T) {
	ts, z, _, err := zookeeper.NewMockZookeeperClient("test", 15*time.Second)
	assert.NoError(t, err)
	err = z.Create("/dubbo/config/dubbo/test-condition.condition-router")
	assert.NoError(t, err)

	data := `enabled: true
force: true
runtime: false
conditions:
  - => host != 172.22.3.91
`

	_, err = z.Conn.Set("/dubbo/config/dubbo/test-condition.condition-router", []byte(data), 0)
	assert.NoError(t, err)
	defer ts.Stop()
	defer z.Close()

	//t.Log(z.Conn.Server())

	zkUrl, _ := common.NewURL(context.TODO(), "zookeeper://127.0.0.1:"+strconv.Itoa(ts.Servers[0].Port))
	configuration, err := extension.GetConfigCenterFactory("zookeeper").GetDynamicConfiguration(&zkUrl)
	config.GetEnvInstance().SetDynamicConfiguration(configuration)

	assert.Nil(t, err)
	assert.NotNil(t, configuration)

	chain := NewRouterChain(getRouteUrl("test-condition"))
	t.Log(chain.routers)
}

func getRouteUrl(applicationKey string) *common.URL {
	url, _ := common.NewURL(context.TODO(), "condition://0.0.0.0/com.foo.BarService")
	url.AddParam("application", applicationKey)
	url.AddParam("force", "true")
	return &url
}
