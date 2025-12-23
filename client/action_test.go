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
		require.Equal(t, "", refOpts.Reference.URL)
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
