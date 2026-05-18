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
	"net/url"
	"sync"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type testConfigurator struct {
	url        *common.URL
	configured bool
}

func (c *testConfigurator) GetUrl() *common.URL {
	return c.url
}

func (c *testConfigurator) Configure(_ *common.URL) {
	c.configured = true
}

type noopTestConfigurator struct {
	url *common.URL
}

func (c *noopTestConfigurator) GetUrl() *common.URL {
	return c.url
}

func (*noopTestConfigurator) Configure(*common.URL) {}

func TestToConfigurators(t *testing.T) {
	makeConfigurator := func(u *common.URL) config_center.Configurator {
		return &testConfigurator{url: u}
	}

	assert.Nil(t, ToConfigurators(nil, makeConfigurator))

	emptyProtocolURL := common.NewURLWithOptions(common.WithProtocol(constant.EmptyProtocol))
	assert.Empty(t, ToConfigurators([]*common.URL{emptyProtocolURL}, makeConfigurator))

	anyhostOnly := common.NewURLWithOptions(common.WithParams(url.Values{
		constant.AnyhostKey: {"true"},
	}))
	assert.Nil(t, ToConfigurators([]*common.URL{anyhostOnly}, makeConfigurator))

	validURL := common.NewURLWithOptions(common.WithProtocol("override"), common.WithParams(url.Values{
		constant.AnyhostKey: {"true"},
		"timeout":           {"2s"},
	}))
	configurators := ToConfigurators([]*common.URL{validURL}, makeConfigurator)
	assert.Len(t, configurators, 1)
	assert.Equal(t, validURL, configurators[0].GetUrl())
}

func TestBaseConfigurationListenerOverrideUrl(t *testing.T) {
	cfg := &testConfigurator{}
	bcl := &BaseConfigurationListener{configurators: []config_center.Configurator{cfg}}
	target := common.NewURLWithOptions()

	bcl.OverrideUrl(target)
	assert.True(t, cfg.configured)
}

func TestBaseConfigurationListenerConfiguratorsSnapshot(t *testing.T) {
	cfg := &testConfigurator{}
	bcl := &BaseConfigurationListener{configurators: []config_center.Configurator{cfg}}

	configurators := bcl.Configurators()
	configurators[0] = &testConfigurator{}

	assert.Equal(t, cfg, bcl.Configurators()[0])
}

func TestBaseConfigurationListenerConfiguratorsConcurrentAccess(t *testing.T) {
	t.Run("should keep configurator snapshots safe if config updates concurrently", func(t *testing.T) {
		dynamicConfiguration := &config_center.MockDynamicConfiguration{}
		dynamicConfiguration.SetParser(&parser.DefaultConfigurationParser{})
		bcl := &BaseConfigurationListener{
			dynamicConfiguration: dynamicConfiguration,
			defaultConfiguratorFunc: func(u *common.URL) config_center.Configurator {
				return &noopTestConfigurator{url: u}
			},
		}
		rawConfig := `configVersion: 2.7.1
scope: notApplication
key: groupA/test:1
enabled: true
configs:
- type: application
  enabled: true
  addresses:
  - 0.0.0.0
  providerAddresses: []
  services:
  - org.apache.dubbo-go.mockService
  applications: []
  parameters:
    cluster: mock1
  side: provider`
		target := common.NewURLWithOptions()

		require.NoError(t, bcl.genConfiguratorFromRawRule(rawConfig))
		assert.Len(t, bcl.Configurators(), 1)

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(4)
			go func() {
				defer wg.Done()
				bcl.Process(&config_center.ConfigChangeEvent{
					ConfigType: remoting.EventTypeUpdate,
					Value:      rawConfig,
				})
			}()
			go func() {
				defer wg.Done()
				bcl.Process(&config_center.ConfigChangeEvent{
					ConfigType: remoting.EventTypeDel,
				})
			}()
			go func() {
				defer wg.Done()
				_ = bcl.Configurators()
			}()
			go func() {
				defer wg.Done()
				bcl.OverrideUrl(target)
			}()
		}
		wg.Wait()
	})
}
