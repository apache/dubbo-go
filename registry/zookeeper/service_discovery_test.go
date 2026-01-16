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

package zookeeper

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
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func Test_newZookeeperServiceDiscovery(t *testing.T) {
	url, _ := common.NewURL("dubbo://127.0.0.1:2181",
		common.WithParamsValue(constant.ClientNameKey, "zk-client"))
	sd, err := newZookeeperServiceDiscovery(url)
	require.NoError(t, err)
	err = sd.Destroy()
	require.NoError(t, err)

}
func Test_zookeeperServiceDiscovery_DataChange(t *testing.T) {
	serviceDiscovery := &zookeeperServiceDiscovery{}
	assert.Equal(t, registry.DefaultPageSize, serviceDiscovery.GetDefaultPageSize())
}
