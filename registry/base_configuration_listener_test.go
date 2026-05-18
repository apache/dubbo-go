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
	"errors"
	"net/url"
	"sync"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	commonConfig "dubbo.apache.org/dubbo-go/v3/common/config"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

const testRawConfiguratorConfig = `configVersion: 2.7.1
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

type testDynamicConfiguration struct {
	parser parser.ConfigurationParser
	raw    string
	err    error
}

func (c *testDynamicConfiguration) Parser() parser.ConfigurationParser {
	return c.parser
}

func (c *testDynamicConfiguration) SetParser(p parser.ConfigurationParser) {
	c.parser = p
}

func (c *testDynamicConfiguration) AddListener(string, config_center.ConfigurationListener, ...config_center.Option) {
}

func (c *testDynamicConfiguration) RemoveListener(string, config_center.ConfigurationListener, ...config_center.Option) {
}

func (c *testDynamicConfiguration) GetProperties(string, ...config_center.Option) (string, error) {
	return c.raw, c.err
}

func (c *testDynamicConfiguration) GetRule(string, ...config_center.Option) (string, error) {
	return c.raw, c.err
}

func (c *testDynamicConfiguration) GetInternalProperty(string, ...config_center.Option) (string, error) {
	return c.raw, c.err
}

func (*testDynamicConfiguration) PublishConfig(string, string, string) error {
	return nil
}

func (*testDynamicConfiguration) RemoveConfig(string, string) error {
	return nil
}

func (*testDynamicConfiguration) GetConfigKeysByGroup(string) (*gxset.HashSet, error) {
	return gxset.NewSet(), nil
}

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

func TestBaseConfigurationListenerInitWith(t *testing.T) {
	t.Run("should clear configurators if dynamic configuration is missing", func(t *testing.T) {
		resetDynamicConfiguration(t, nil)
		cfg := &testConfigurator{}
		bcl := &BaseConfigurationListener{configurators: []config_center.Configurator{cfg}}

		bcl.InitWith("key", bcl, newNoopConfigurator)

		assert.Empty(t, bcl.Configurators())
	})

	t.Run("should clear configurators if initial rule lookup fails", func(t *testing.T) {
		resetDynamicConfiguration(t, &testDynamicConfiguration{err: errors.New("rule lookup failed")})
		cfg := &testConfigurator{}
		bcl := &BaseConfigurationListener{configurators: []config_center.Configurator{cfg}}

		bcl.InitWith("key", bcl, newNoopConfigurator)

		assert.Empty(t, bcl.Configurators())
	})

	t.Run("should initialize configurators if initial rule exists", func(t *testing.T) {
		resetDynamicConfiguration(t, &testDynamicConfiguration{
			parser: &parser.DefaultConfigurationParser{},
			raw:    testRawConfiguratorConfig,
		})
		bcl := &BaseConfigurationListener{}

		bcl.InitWith("key", bcl, newNoopConfigurator)

		assert.Len(t, bcl.Configurators(), 1)
	})

	t.Run("should ignore initial rule if it cannot be parsed", func(t *testing.T) {
		resetDynamicConfiguration(t, &testDynamicConfiguration{
			parser: &parser.DefaultConfigurationParser{},
			raw:    "invalid: [",
		})
		bcl := &BaseConfigurationListener{}

		bcl.InitWith("key", bcl, newNoopConfigurator)

		assert.Nil(t, bcl.Configurators())
	})
}

func TestBaseConfigurationListenerProcessInvalidConfig(t *testing.T) {
	t.Run("should keep configurators unchanged if update config cannot be parsed", func(t *testing.T) {
		cfg := &testConfigurator{}
		bcl := &BaseConfigurationListener{
			configurators:           []config_center.Configurator{cfg},
			dynamicConfiguration:    &testDynamicConfiguration{parser: &parser.DefaultConfigurationParser{}},
			defaultConfiguratorFunc: newNoopConfigurator,
		}

		bcl.Process(&config_center.ConfigChangeEvent{
			ConfigType: remoting.EventTypeUpdate,
			Value:      "invalid: [",
		})

		assert.Equal(t, cfg, bcl.Configurators()[0])
	})
}

func TestBaseConfigurationListenerConfiguratorsConcurrentAccess(t *testing.T) {
	t.Run("should keep configurator snapshots safe if config updates concurrently", func(t *testing.T) {
		dynamicConfiguration := &config_center.MockDynamicConfiguration{}
		dynamicConfiguration.SetParser(&parser.DefaultConfigurationParser{})
		bcl := &BaseConfigurationListener{
			dynamicConfiguration:    dynamicConfiguration,
			defaultConfiguratorFunc: newNoopConfigurator,
		}
		target := common.NewURLWithOptions()

		require.NoError(t, bcl.genConfiguratorFromRawRule(testRawConfiguratorConfig))
		assert.Len(t, bcl.Configurators(), 1)

		var wg sync.WaitGroup
		for i := 0; i < 100; i++ {
			wg.Add(4)
			go func() {
				defer wg.Done()
				bcl.Process(&config_center.ConfigChangeEvent{
					ConfigType: remoting.EventTypeUpdate,
					Value:      testRawConfiguratorConfig,
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

func newNoopConfigurator(u *common.URL) config_center.Configurator {
	return &noopTestConfigurator{url: u}
}

func resetDynamicConfiguration(t *testing.T, dc config_center.DynamicConfiguration) {
	t.Helper()
	env := commonConfig.GetEnvInstance()
	previous := env.GetDynamicConfiguration()
	env.SetDynamicConfiguration(dc)
	t.Cleanup(func() {
		env.SetDynamicConfiguration(previous)
	})
}
