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

package config_center

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/constant/file"
	"dubbo.apache.org/dubbo-go/v3/global"
)

func TestOptionsProtocolAndAddress(t *testing.T) {
	cases := []struct {
		name     string
		opts     []Option
		expected string
	}{
		{name: "zookeeper", opts: []Option{WithZookeeper()}, expected: constant.ZookeeperKey},
		{name: "nacos", opts: []Option{WithNacos()}, expected: constant.NacosKey},
		{name: "apollo", opts: []Option{WithApollo()}, expected: constant.ApolloKey},
		{name: "custom", opts: []Option{WithConfigCenter("etcd")}, expected: "etcd"},
		{name: "address with scheme", opts: []Option{WithAddress("zk://127.0.0.1:2181")}, expected: "zk"},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			co := NewOptions(tt.opts...)
			assert.Equal(t, tt.expected, co.Center.Protocol)
		})
	}
}

func TestOptionsFields(t *testing.T) {
	center := &global.CenterConfig{}
	opts := []Option{
		WithDataID("dataId"),
		WithCluster("cluster"),
		WithGroup("group"),
		WithUsername("user"),
		WithPassword("pwd"),
		WithNamespace("ns"),
		WithAppID("app"),
		WithTimeout(1500 * time.Millisecond),
		WithParams(map[string]string{"k": "v"}),
		WithFile(),
		WithFileExtJson(),
	}

	wrapped := &Options{Center: center}
	for _, o := range opts {
		o(wrapped)
	}

	assert.Equal(t, "dataId", center.DataId)
	assert.Equal(t, "cluster", center.Cluster)
	assert.Equal(t, "group", center.Group)
	assert.Equal(t, "user", center.Username)
	assert.Equal(t, "pwd", center.Password)
	assert.Equal(t, "ns", center.Namespace)
	assert.Equal(t, "app", center.AppID)
	assert.Equal(t, "1500", center.Timeout)
	assert.Equal(t, map[string]string{"k": "v"}, center.Params)
	assert.Equal(t, constant.FileKey, center.Protocol)
	assert.Equal(t, string(file.JSON), center.FileExtension)
}
