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

package server

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/global"
)

// Test compatApplicationConfig with all fields
func TestCompatApplicationConfigAll(t *testing.T) {
	appCfg := &global.ApplicationConfig{
		Organization:            "test-org",
		Name:                    "test-app",
		Module:                  "test-module",
		Group:                   "test-group",
		Version:                 "1.0.0",
		Owner:                   "test-owner",
		Environment:             "test-env",
		MetadataType:            "remote",
		Tag:                     "v1",
		MetadataServicePort:     "30880",
		MetadataServiceProtocol: "triple",
	}

	result := compatApplicationConfig(appCfg)

	assert.NotNil(t, result)
	assert.Equal(t, "test-org", result.Organization)
	assert.Equal(t, "test-app", result.Name)
	assert.Equal(t, "test-module", result.Module)
	assert.Equal(t, "test-group", result.Group)
	assert.Equal(t, "1.0.0", result.Version)
	assert.Equal(t, "test-owner", result.Owner)
	assert.Equal(t, "test-env", result.Environment)
	assert.Equal(t, "remote", result.MetadataType)
	assert.Equal(t, "v1", result.Tag)
	assert.Equal(t, "30880", result.MetadataServicePort)
	assert.Equal(t, "triple", result.MetadataServiceProtocol)
}

// Test compatApplicationConfig with partial fields
func TestCompatApplicationConfigPartial(t *testing.T) {
	appCfg := &global.ApplicationConfig{
		Name:    "test-app",
		Version: "1.0.0",
	}

	result := compatApplicationConfig(appCfg)

	assert.NotNil(t, result)
	assert.Equal(t, "test-app", result.Name)
	assert.Equal(t, "1.0.0", result.Version)
	assert.Empty(t, result.Organization)
	assert.Empty(t, result.Module)
}

// Test compatApplicationConfig with empty config
func TestCompatApplicationConfigEmpty(t *testing.T) {
	appCfg := &global.ApplicationConfig{}

	result := compatApplicationConfig(appCfg)

	assert.NotNil(t, result)
	assert.Empty(t, result.Name)
	assert.Empty(t, result.Organization)
	assert.Empty(t, result.Module)
	assert.Empty(t, result.Group)
	assert.Empty(t, result.Version)
	assert.Empty(t, result.Owner)
	assert.Empty(t, result.Environment)
	assert.Empty(t, result.MetadataType)
	assert.Empty(t, result.Tag)
	assert.Empty(t, result.MetadataServicePort)
	assert.Empty(t, result.MetadataServiceProtocol)
}

// Test compatRegistryConfig with all fields
func TestCompatRegistryConfigAll(t *testing.T) {
	regCfg := &global.RegistryConfig{
		Protocol:          "zookeeper",
		Timeout:           "10s",
		Group:             "registry-group",
		Namespace:         "test-ns",
		TTL:               "60s",
		Address:           "localhost:2181",
		Username:          "admin",
		Password:          "password",
		Simplified:        true,
		Preferred:         true,
		Zone:              "zone1",
		Weight:            100,
		Params:            map[string]string{"key": "value"},
		RegistryType:      "service_discovery",
		UseAsMetaReport:   "true",
		UseAsConfigCenter: "true",
	}

	result := compatRegistryConfig(regCfg)

	assert.NotNil(t, result)
	assert.Equal(t, "zookeeper", result.Protocol)
	assert.Equal(t, "10s", result.Timeout)
	assert.Equal(t, "registry-group", result.Group)
	assert.Equal(t, "test-ns", result.Namespace)
	assert.Equal(t, "60s", result.TTL)
	assert.Equal(t, "localhost:2181", result.Address)
	assert.Equal(t, "admin", result.Username)
	assert.Equal(t, "password", result.Password)
	assert.True(t, result.Simplified)
	assert.True(t, result.Preferred)
	assert.Equal(t, "zone1", result.Zone)
	assert.Equal(t, int64(100), result.Weight)
	assert.Equal(t, "service_discovery", result.RegistryType)
	assert.Equal(t, "true", result.UseAsMetaReport)
	assert.Equal(t, "true", result.UseAsConfigCenter)
	assert.NotNil(t, result.Params)
	assert.Equal(t, "value", result.Params["key"])
}

// Test compatRegistryConfig with partial fields
func TestCompatRegistryConfigPartial(t *testing.T) {
	regCfg := &global.RegistryConfig{
		Protocol: "nacos",
		Address:  "localhost:8848",
	}

	result := compatRegistryConfig(regCfg)

	assert.NotNil(t, result)
	assert.Equal(t, "nacos", result.Protocol)
	assert.Equal(t, "localhost:8848", result.Address)
	assert.Empty(t, result.Timeout)
	assert.Empty(t, result.Group)
}

// Test compatRegistryConfig with empty config
func TestCompatRegistryConfigEmpty(t *testing.T) {
	regCfg := &global.RegistryConfig{}

	result := compatRegistryConfig(regCfg)

	assert.NotNil(t, result)
	assert.Empty(t, result.Protocol)
	assert.Empty(t, result.Address)
	assert.Empty(t, result.Timeout)
	assert.Empty(t, result.Group)
	assert.Empty(t, result.Namespace)
	assert.Empty(t, result.TTL)
	assert.Empty(t, result.Username)
	assert.Empty(t, result.Password)
	assert.False(t, result.Simplified)
	assert.False(t, result.Preferred)
}

// Test compatRegistryConfig with boolean fields
func TestCompatRegistryConfigBooleans(t *testing.T) {
	regCfg := &global.RegistryConfig{
		Protocol:          "etcd",
		Simplified:        true,
		Preferred:         true,
		UseAsMetaReport:   "true",
		UseAsConfigCenter: "false",
	}

	result := compatRegistryConfig(regCfg)

	assert.NotNil(t, result)
	assert.True(t, result.Simplified)
	assert.True(t, result.Preferred)
	assert.Equal(t, "true", result.UseAsMetaReport)
	assert.Equal(t, "false", result.UseAsConfigCenter)
}

// Test compatRegistryConfig with numeric fields
func TestCompatRegistryConfigNumeric(t *testing.T) {
	regCfg := &global.RegistryConfig{
		Protocol: "zookeeper",
		Weight:   500,
	}

	result := compatRegistryConfig(regCfg)

	assert.NotNil(t, result)
	assert.Equal(t, "zookeeper", result.Protocol)
	assert.Equal(t, int64(500), result.Weight)
}

// Test compatRegistryConfig with params
func TestCompatRegistryConfigWithParams(t *testing.T) {
	regCfg := &global.RegistryConfig{
		Protocol: "zookeeper",
		Params: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	result := compatRegistryConfig(regCfg)

	assert.NotNil(t, result)
	assert.Equal(t, 3, len(result.Params))
	assert.Equal(t, "value1", result.Params["key1"])
	assert.Equal(t, "value2", result.Params["key2"])
	assert.Equal(t, "value3", result.Params["key3"])
}

// Test compatApplicationConfig preserves all field types
func TestCompatApplicationConfigFieldPreservation(t *testing.T) {
	appCfg := &global.ApplicationConfig{
		Name:                    "myapp",
		Organization:            "myorg",
		Module:                  "mymodule",
		Group:                   "mygroup",
		Version:                 "2.0.0",
		Owner:                   "owner",
		Environment:             "production",
		MetadataType:            "local",
		Tag:                     "beta",
		MetadataServicePort:     "25555",
		MetadataServiceProtocol: "grpc",
	}

	result := compatApplicationConfig(appCfg)

	// Verify type conversion doesn't lose data
	assert.IsType(t, (*config.ApplicationConfig)(nil), result)
	assert.Equal(t, appCfg.Name, result.Name)
	assert.Equal(t, appCfg.Organization, result.Organization)
	assert.Equal(t, appCfg.Module, result.Module)
}

// Test compatRegistryConfig preserves all field types
func TestCompatRegistryConfigFieldPreservation(t *testing.T) {
	regCfg := &global.RegistryConfig{
		Protocol:          "consul",
		Timeout:           "30s",
		Group:             "consul-group",
		Namespace:         "consul-ns",
		TTL:               "120s",
		Address:           "localhost:8500",
		Username:          "consul-user",
		Password:          "consul-pass",
		Simplified:        true,
		Preferred:         false,
		Zone:              "zone-a",
		Weight:            200,
		Params:            map[string]string{"consul-key": "consul-val"},
		RegistryType:      "interface",
		UseAsMetaReport:   "true",
		UseAsConfigCenter: "false",
	}

	result := compatRegistryConfig(regCfg)

	// Verify type conversion doesn't lose data
	assert.IsType(t, (*config.RegistryConfig)(nil), result)
	assert.Equal(t, regCfg.Protocol, result.Protocol)
	assert.Equal(t, regCfg.Timeout, result.Timeout)
	assert.Equal(t, regCfg.Group, result.Group)
}

// Test multiple conversions don't affect original
func TestCompatConfigsPreservesOriginal(t *testing.T) {
	appCfg := &global.ApplicationConfig{
		Name:    "original-app",
		Version: "1.0.0",
	}

	result1 := compatApplicationConfig(appCfg)
	result2 := compatApplicationConfig(appCfg)

	assert.Equal(t, "original-app", appCfg.Name)
	assert.Equal(t, "1.0.0", appCfg.Version)

	assert.Equal(t, result1.Name, result2.Name)
	assert.Equal(t, result1.Version, result2.Version)
	assert.Equal(t, "original-app", result1.Name)
}

// Test compatRegistryConfig with special characters in params
func TestCompatRegistryConfigSpecialChars(t *testing.T) {
	regCfg := &global.RegistryConfig{
		Protocol: "zookeeper",
		Params: map[string]string{
			"special.key":    "value with spaces",
			"another-key":    "value/with/slashes",
			"key_with_colon": "value:with:colons",
			"key.with.dots":  "value.with.dots",
		},
	}

	result := compatRegistryConfig(regCfg)

	assert.NotNil(t, result)
	assert.Equal(t, "value with spaces", result.Params["special.key"])
	assert.Equal(t, "value/with/slashes", result.Params["another-key"])
	assert.Equal(t, "value:with:colons", result.Params["key_with_colon"])
	assert.Equal(t, "value.with.dots", result.Params["key.with.dots"])
}
