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
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config_center"
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
