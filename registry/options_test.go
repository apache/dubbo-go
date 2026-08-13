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

package registry

import (
	"testing"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"github.com/stretchr/testify/assert"
)

func TestNewOptionsRequireProtocol(t *testing.T) {
	assert.Panics(t, func() {
		NewOptions()
	})
}

func testRegistryConfig(protocol string, set ...func(*global.RegistryConfig)) *global.RegistryConfig {
	cfg := global.DefaultRegistryConfig()
	cfg.Protocol = protocol
	for _, f := range set {
		f(cfg)
	}
	return cfg
}

func TestNewOptionsWithHelpers(t *testing.T) {
	tests := []struct {
		name   string
		opts   []Option
		wantID string
		want   *global.RegistryConfig
	}{
		{
			name:   "zookeeper default id",
			opts:   []Option{WithZookeeper()},
			wantID: constant.ZookeeperKey,
			want:   testRegistryConfig(constant.ZookeeperKey),
		},

		{
			name:   "etcd with custom id",
			opts:   []Option{WithEtcdV3(), WithID("custom-id")},
			wantID: "custom-id",
			want:   testRegistryConfig(constant.EtcdV3Key),
		},

		{
			name:   "nacos protocol",
			opts:   []Option{WithNacos()},
			wantID: constant.NacosKey,
			want:   testRegistryConfig(constant.NacosKey),
		},

		{
			name:   "polaris protocol",
			opts:   []Option{WithPolaris()},
			wantID: constant.PolarisKey,
			want:   testRegistryConfig(constant.PolarisKey),
		},

		{
			name:   "registry by name",
			opts:   []Option{WithRegistry(constant.ZookeeperKey)},
			wantID: constant.ZookeeperKey,
			want:   testRegistryConfig(constant.ZookeeperKey),
		},

		{
			name:   "address overrides protocol",
			opts:   []Option{WithAddress("nacos://127.0.0.1:8848")},
			wantID: constant.NacosKey,
			want: testRegistryConfig(constant.NacosKey, func(c *global.RegistryConfig) {
				c.Address = "nacos://127.0.0.1:8848"
			}),
		},

		{
			name:   "address without scheme",
			opts:   []Option{WithZookeeper(), WithAddress("127.0.0.1:2181")},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.Address = "127.0.0.1:2181"
			}),
		},

		{
			name:   "timeout option",
			opts:   []Option{WithZookeeper(), WithTimeout(3 * time.Second)},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.Timeout = "3s"
			}),
		},

		{
			name:   "ttl option",
			opts:   []Option{WithZookeeper(), WithTTL(30 * time.Minute)},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.TTL = "30m0s"
			}),
		},

		{
			name:   "group option",
			opts:   []Option{WithZookeeper(), WithGroup("dev")},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.Group = "dev"
			}),
		},

		{
			name:   "namespace option",
			opts:   []Option{WithZookeeper(), WithNamespace("ns")},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.Namespace = "ns"
			}),
		},

		{
			name:   "username and password",
			opts:   []Option{WithZookeeper(), WithUsername("user"), WithPassword("pass")},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.Username = "user"
				c.Password = "pass"
			}),
		},

		{
			name:   "simplified option",
			opts:   []Option{WithZookeeper(), WithSimplified()},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.Simplified = true
			}),
		},

		{
			name:   "preferred option",
			opts:   []Option{WithZookeeper(), WithPreferred()},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.Preferred = true
			}),
		},

		{
			name:   "zone option",
			opts:   []Option{WithZookeeper(), WithZone("zone-a")},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.Zone = "zone-a"
			}),
		},

		{
			name:   "weight option",
			opts:   []Option{WithZookeeper(), WithWeight(100)},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.Weight = 100
			}),
		},

		{
			name:   "params option",
			opts:   []Option{WithZookeeper(), WithParams(map[string]string{"key1": "value1", "key2": "value2"})},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.Params = map[string]string{"key1": "value1", "key2": "value2"}
			}),
		},

		{
			name:   "register service and interface",
			opts:   []Option{WithZookeeper(), WithRegisterServiceAndInterface()},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.RegistryType = constant.RegistryTypeAll
			}),
		},

		{
			name:   "register interface only",
			opts:   []Option{WithZookeeper(), WithRegisterInterface()},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.RegistryType = constant.RegistryTypeInterface
			}),
		},

		{
			name:   "register service only",
			opts:   []Option{WithZookeeper(), WithRegisterService()},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.RegistryType = constant.RegistryTypeService
			}),
		},

		{
			name:   "not used as meta report",
			opts:   []Option{WithZookeeper(), WithoutUseAsMetaReport()},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.UseAsMetaReport = "false"
			}),
		},

		{
			name:   "not used as config center",
			opts:   []Option{WithZookeeper(), WithoutUseAsConfigCenter()},
			wantID: constant.ZookeeperKey,
			want: testRegistryConfig(constant.ZookeeperKey, func(c *global.RegistryConfig) {
				c.UseAsConfigCenter = "false"
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := NewOptions(tt.opts...)
			assert.Equal(t, tt.want, options.Registry)
			assert.Equal(t, tt.wantID, options.ID)
		})
	}
}
