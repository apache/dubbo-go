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

package client

import (
	"os"
	"strconv"
	"testing"
)

import (
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

func TestGetEnv(t *testing.T) {
	key := "TEST_GET_ENV_KEY"
	_ = os.Unsetenv(key)
	require.Equal(t, "fallback", getEnv(key, "fallback"))

	require.NoError(t, os.Setenv(key, "value"))
	defer os.Unsetenv(key)
	require.Equal(t, "value", getEnv(key, "fallback"))
}

func TestUpdateOrCreateMeshURL(t *testing.T) {
	t.Run("mesh disabled", func(t *testing.T) {
		refOpts := &ReferenceOptions{
			Reference: &global.ReferenceConfig{URL: ""},
			Consumer:  &global.ConsumerConfig{MeshEnabled: false},
		}
		updateOrCreateMeshURL(refOpts)
		require.Empty(t, refOpts.Reference.URL)
	})

	t.Run("mesh enabled build url", func(t *testing.T) {
		defer os.Unsetenv(constant.PodNamespaceEnvKey)
		defer os.Unsetenv(constant.ClusterDomainKey)
		require.NoError(t, os.Setenv(constant.PodNamespaceEnvKey, "ns"))
		require.NoError(t, os.Setenv(constant.ClusterDomainKey, "cluster.local"))

		refOpts := &ReferenceOptions{
			Reference: &global.ReferenceConfig{
				Protocol:         constant.TriProtocol,
				ProvidedBy:       "svc",
				MeshProviderPort: 0,
			},
			Consumer: &global.ConsumerConfig{MeshEnabled: true},
		}
		updateOrCreateMeshURL(refOpts)
		expected := "tri://svc.ns" + constant.SVC + "cluster.local:" + strconv.Itoa(constant.DefaultMeshPort)
		require.Equal(t, expected, refOpts.Reference.URL)
	})

	t.Run("panic non tri protocol", func(t *testing.T) {
		refOpts := &ReferenceOptions{
			Reference: &global.ReferenceConfig{
				Protocol:   "dubbo",
				ProvidedBy: "svc",
			},
			Consumer: &global.ConsumerConfig{MeshEnabled: true},
		}
		require.Panics(t, func() {
			updateOrCreateMeshURL(refOpts)
		})
	})

	t.Run("panic empty providedBy", func(t *testing.T) {
		refOpts := &ReferenceOptions{
			Reference: &global.ReferenceConfig{
				Protocol:   constant.TriProtocol,
				ProvidedBy: "",
			},
			Consumer: &global.ConsumerConfig{MeshEnabled: true},
		}
		require.Panics(t, func() {
			updateOrCreateMeshURL(refOpts)
		})
	})
}

func TestProcessURLPropagatesRegistryError(t *testing.T) {
	ref := &global.ReferenceConfig{
		InterfaceName: "com.example.Service",
		RegistryIDs:   []string{"bad"},
	}
	cfgURL := common.NewURLWithOptions(common.WithPath(ref.InterfaceName))
	registries := map[string]*global.RegistryConfig{
		"bad": {
			Protocol: "mock",
			Address:  "127.0.0.1:bad",
		},
	}

	urls, err := processURL(ref, registries, cfgURL)
	require.Nil(t, urls)
	require.Error(t, err)
	require.Contains(t, err.Error(), `registry id "bad" url is invalid`)
}

func TestProcessURLRejectsEmptyRegistryURLs(t *testing.T) {
	ref := &global.ReferenceConfig{
		InterfaceName: "com.example.Service",
		RegistryIDs:   []string{"empty"},
	}
	cfgURL := common.NewURLWithOptions(common.WithPath(ref.InterfaceName))
	registries := map[string]*global.RegistryConfig{
		"empty": {
			Protocol: "mock",
			Address:  constant.NotAvailable,
		},
	}

	urls, err := processURL(ref, registries, cfgURL)
	require.Nil(t, urls)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no registry urls available")
}

func TestProcessURLWithDirectUserURL(t *testing.T) {
	ref := &global.ReferenceConfig{
		InterfaceName: "com.example.Service",
		Protocol:      constant.TriProtocol,
		URL:           "tri://localhost:20000",
	}
	cfgURL := common.NewURLWithOptions(
		common.WithPath(ref.InterfaceName),
		common.WithParamsValue(constant.GroupKey, "test-group"),
	)

	urls, err := processURL(ref, nil, cfgURL)
	require.NoError(t, err)
	require.Len(t, urls, 1)
	require.Equal(t, constant.TriProtocol, urls[0].Protocol)
	require.Equal(t, "/"+ref.InterfaceName, urls[0].Path)
	require.Equal(t, "test-group", urls[0].GetParam(constant.GroupKey, ""))
	require.Equal(t, "true", urls[0].GetParam("peer", ""))
}

func TestProcessURLWithMultipleDirectUserURLs(t *testing.T) {
	ref := &global.ReferenceConfig{
		InterfaceName: "com.example.Service",
		Protocol:      constant.TriProtocol,
		URL:           "tri://localhost:20000;tri://localhost:20001",
	}
	cfgURL := common.NewURLWithOptions(common.WithPath(ref.InterfaceName))

	urls, err := processURL(ref, nil, cfgURL)
	require.NoError(t, err)
	require.Len(t, urls, 2)
	require.Equal(t, constant.TriProtocol, urls[0].Protocol)
	require.Equal(t, "true", urls[0].GetParam("peer", ""))
	require.Equal(t, constant.TriProtocol, urls[1].Protocol)
	require.Equal(t, "true", urls[1].GetParam("peer", ""))
}

func TestProcessURLRejectsInvalidUserURL(t *testing.T) {
	ref := &global.ReferenceConfig{
		InterfaceName: "com.example.Service",
		Protocol:      constant.TriProtocol,
		URL:           "://bad-url",
	}
	cfgURL := common.NewURLWithOptions(common.WithPath(ref.InterfaceName))

	urls, err := processURL(ref, nil, cfgURL)
	require.Nil(t, urls)
	require.Error(t, err)
	require.Contains(t, err.Error(), "url configuration error")
}

func TestBuildReferenceURLWithRegistryProtocol(t *testing.T) {
	ref := &global.ReferenceConfig{InterfaceName: "com.example.Service"}
	cfgURL := common.NewURLWithOptions(common.WithPath(ref.InterfaceName))
	serviceURL := common.NewURLWithOptions(common.WithProtocol(constant.RegistryProtocol))

	result := buildReferenceURL(serviceURL, ref, cfgURL)

	require.Same(t, serviceURL, result)
	require.Same(t, cfgURL, result.SubURL)
}

func TestLoadRegistryURLsAssignsSubURL(t *testing.T) {
	ref := &global.ReferenceConfig{
		InterfaceName: "com.example.Service",
		RegistryIDs:   []string{"valid"},
	}
	cfgURL := common.NewURLWithOptions(common.WithPath(ref.InterfaceName))
	registries := map[string]*global.RegistryConfig{
		"valid": {
			Protocol: "zookeeper",
			Address:  "127.0.0.1:2181",
		},
	}

	urls, err := loadRegistryURLs(ref, registries, cfgURL)
	require.NoError(t, err)
	require.NotEmpty(t, urls)
	for _, regURL := range urls {
		require.Same(t, cfgURL, regURL.SubURL)
	}
}

func TestBuildInvokerRejectsEmptyURLs(t *testing.T) {
	invoker, err := buildInvoker(nil, &global.ReferenceConfig{})
	require.Nil(t, invoker)
	require.Error(t, err)
	require.Contains(t, err.Error(), "no urls available")
}
