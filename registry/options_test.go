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
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestNewOptionsRequireProtocol(t *testing.T) {
	assert.Panics(t, func() {
		NewOptions()
	})
}

func TestNewOptionsWithHelpers(t *testing.T) {
	tests := []struct {
		name         string
		opts         []Option
		wantProtocol string
		wantID       string
		wantTimeout  string
		wantAddress  string
	}{
		{
			name:         "zookeeper default id",
			opts:         []Option{WithZookeeper()},
			wantProtocol: constant.ZookeeperKey,
			wantID:       constant.ZookeeperKey,
		},
		{
			name:         "etcd with custom id",
			opts:         []Option{WithEtcdV3(), WithID("custom-id")},
			wantProtocol: constant.EtcdV3Key,
			wantID:       "custom-id",
		},
		{
			name:         "address overrides protocol",
			opts:         []Option{WithAddress("nacos://127.0.0.1:8848")},
			wantProtocol: constant.NacosKey,
			wantID:       constant.NacosKey,
			wantAddress:  "nacos://127.0.0.1:8848",
		},
		{
			name:         "timeout option",
			opts:         []Option{WithZookeeper(), WithTimeout(3 * time.Second)},
			wantProtocol: constant.ZookeeperKey,
			wantID:       constant.ZookeeperKey,
			wantTimeout:  "3s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			options := NewOptions(tt.opts...)
			assert.Equal(t, tt.wantProtocol, options.Registry.Protocol)
			assert.Equal(t, tt.wantID, options.ID)
			if tt.wantTimeout != "" {
				assert.Equal(t, tt.wantTimeout, options.Registry.Timeout)
			}
			if tt.wantAddress != "" {
				assert.Equal(t, tt.wantAddress, options.Registry.Address)
			}
		})
	}
}
