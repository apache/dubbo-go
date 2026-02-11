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

package internal

import (
	"strconv"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

func TestLoadRegistries_AllOrEmptyIDs(t *testing.T) {
	registries := map[string]*global.RegistryConfig{
		"r1": {
			Protocol:     "mock",
			Timeout:      "2s",
			Group:        "g1",
			Address:      "127.0.0.1:2181",
			RegistryType: constant.RegistryTypeInterface,
		},
		"r2": {
			Protocol:     "mock",
			Timeout:      "2s",
			Group:        "g2",
			Address:      "127.0.0.2:2181",
			RegistryType: constant.RegistryTypeAll,
		},
	}

	tests := []struct {
		name string
		ids  []string
	}{
		{name: "nil", ids: nil},
		{name: "empty-string", ids: []string{""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			urls := LoadRegistries(tt.ids, registries, common.CONSUMER)
			require.Len(t, urls, 3)

			counts := map[string]int{}
			for _, u := range urls {
				id := u.GetParam(constant.RegistryIdKey, "")
				require.NotEmpty(t, id)
				counts[id]++
			}
			assert.Equal(t, 1, counts["r1"])
			assert.Equal(t, 2, counts["r2"])
		})
	}
}

func TestLoadRegistries_FilteredIDs(t *testing.T) {
	registries := map[string]*global.RegistryConfig{
		"r1": {
			Protocol: "mock",
			Timeout:  "2s",
			Group:    "g1",
			Address:  "127.0.0.1:2181",
		},
		"r2": {
			Protocol: "mock",
			Timeout:  "2s",
			Group:    "g2",
			Address:  "127.0.0.2:2181",
		},
	}

	urls := LoadRegistries([]string{"r1"}, registries, common.CONSUMER)
	require.Len(t, urls, 1)
	for _, u := range urls {
		assert.Equal(t, "r1", u.GetParam(constant.RegistryIdKey, ""))
	}
}

func TestLoadRegistries_PanicOnInvalidURL(t *testing.T) {
	registries := map[string]*global.RegistryConfig{
		"bad": {
			Protocol: "mock",
			Timeout:  "2s",
			Group:    "g1",
			Address:  "127.0.0.1:bad",
		},
	}

	assert.Panics(t, func() {
		_ = LoadRegistries([]string{"bad"}, registries, common.CONSUMER)
	})
}

func TestToURLs_EmptyOrNA(t *testing.T) {
	tests := []struct {
		name    string
		address string
	}{
		{name: "empty", address: ""},
		{name: "na", address: constant.NotAvailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &global.RegistryConfig{
				Protocol: "mock",
				Address:  tt.address,
			}
			urls, err := toURLs(cfg, common.CONSUMER)
			require.NoError(t, err)
			assert.Len(t, urls, 0)
		})
	}
}

func TestToURLs_RegistryTypeAll(t *testing.T) {
	cfg := &global.RegistryConfig{
		Protocol:     "mock",
		Address:      "127.0.0.1:2181",
		RegistryType: constant.RegistryTypeAll,
	}

	urls, err := toURLs(cfg, common.CONSUMER)
	require.NoError(t, err)
	require.Len(t, urls, 2)

	protocols := map[string]bool{}
	for _, u := range urls {
		protocols[u.Protocol] = true
		assert.Equal(t, "127.0.0.1:2181", u.Location)
	}
	assert.True(t, protocols[constant.ServiceRegistryProtocol])
	assert.True(t, protocols[constant.RegistryProtocol])
}

func TestTranslateRegistryAddress(t *testing.T) {
	tests := []struct {
		name      string
		address   string
		want      string
		wantPanic bool
	}{
		{name: "simple", address: "nacos://127.0.0.1:8848", want: "127.0.0.1:8848"},
		{name: "path", address: "nacos://127.0.0.1:8848/path", want: "127.0.0.1:8848/path"},
		{name: "invalid", address: "://bad", wantPanic: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &global.RegistryConfig{Address: tt.address}
			if tt.wantPanic {
				assert.Panics(t, func() {
					_ = translateRegistryAddress(cfg)
				})
				return
			}
			got := translateRegistryAddress(cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestGetUrlMap_BasicAndOverrides(t *testing.T) {
	cfg := &global.RegistryConfig{
		Protocol:     "mock",
		Timeout:      "5s",
		Group:        "group",
		TTL:          "15m",
		Address:      "127.0.0.1:2181",
		Preferred:    true,
		Zone:         "zone",
		Weight:       200,
		RegistryType: constant.RegistryTypeInterface,
		Params: map[string]string{
			"custom":                    "x",
			constant.RegistryTimeoutKey: "override",
		},
	}

	values := getUrlMap(cfg, common.PROVIDER)
	assert.Equal(t, "group", values.Get(constant.RegistryGroupKey))
	assert.Equal(t, strconv.Itoa(int(common.PROVIDER)), values.Get(constant.RegistryRoleKey))
	assert.Equal(t, "mock", values.Get(constant.RegistryKey))
	assert.Equal(t, "override", values.Get(constant.RegistryTimeoutKey))
	assert.Equal(t, "true", values.Get(constant.RegistryKey+"."+constant.RegistryLabelKey))
	assert.Equal(t, "true", values.Get(constant.RegistryKey+"."+constant.PreferredKey))
	assert.Equal(t, "zone", values.Get(constant.RegistryKey+"."+constant.RegistryZoneKey))
	assert.Equal(t, "200", values.Get(constant.RegistryKey+"."+constant.WeightKey))
	assert.Equal(t, "15m", values.Get(constant.RegistryTTLKey))
	assert.Equal(t, "x", values.Get("custom"))
	assert.Equal(t, clientNameID(cfg.Protocol, cfg.Address), values.Get(constant.ClientNameKey))
	assert.Equal(t, constant.RegistryTypeInterface, values.Get(constant.RegistryTypeKey))
}

func TestCreateNewURL_SetsFields(t *testing.T) {
	cfg := &global.RegistryConfig{
		Protocol:     "mock",
		Timeout:      "10s",
		Group:        "group",
		Namespace:    "ns",
		TTL:          "15m",
		Address:      "127.0.0.1:2181",
		Username:     "user",
		Password:     "pass",
		Simplified:   true,
		Preferred:    false,
		Zone:         "zone",
		Weight:       100,
		RegistryType: constant.RegistryTypeService,
		Params: map[string]string{
			"custom": "x",
		},
	}

	u, err := createNewURL(cfg, constant.ServiceRegistryProtocol, cfg.Address, common.CONSUMER)
	require.NoError(t, err)

	assert.Equal(t, constant.ServiceRegistryProtocol, u.Protocol)
	assert.Equal(t, cfg.Address, u.Location)
	assert.Equal(t, cfg.Username, u.Username)
	assert.Equal(t, cfg.Password, u.Password)
	assert.Equal(t, "true", u.GetParam(constant.RegistrySimplifiedKey, ""))
	assert.Equal(t, cfg.Protocol, u.GetParam(constant.RegistryKey, ""))
	assert.Equal(t, cfg.Namespace, u.GetParam(constant.RegistryNamespaceKey, ""))
	assert.Equal(t, cfg.Timeout, u.GetParam(constant.RegistryTimeoutKey, ""))
	assert.Equal(t, strconv.Itoa(int(common.CONSUMER)), u.GetParam(constant.RegistryRoleKey, ""))
	assert.Equal(t, cfg.Group, u.GetParam(constant.RegistryGroupKey, ""))
	assert.Equal(t, cfg.TTL, u.GetParam(constant.RegistryTTLKey, ""))
	assert.Equal(t, clientNameID(cfg.Protocol, cfg.Address), u.GetParam(constant.ClientNameKey, ""))
	assert.Equal(t, "x", u.GetParam("custom", ""))
}

func TestClientNameID(t *testing.T) {
	got := clientNameID("nacos", "127.0.0.1:8848")
	assert.Equal(t, "dubbo.registries-nacos-127.0.0.1:8848", got)
}
