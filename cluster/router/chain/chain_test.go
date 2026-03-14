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

package chain

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
)

// TestIsRouterMatch_ServiceMatch tests service router matching
func TestIsRouterMatch_ServiceMatch(t *testing.T) {
	routerCfg := &global.RouterConfig{
		Scope: "service",
		Key:   "test.service",
	}

	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)
	url.SubURL = common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithPath("test.service"),
	)

	assert.True(t, isRouterMatch(routerCfg, url, ""))
}

// TestIsRouterMatch_ServiceMismatch tests service router mismatch
func TestIsRouterMatch_ServiceMismatch(t *testing.T) {
	routerCfg := &global.RouterConfig{
		Scope: "service",
		Key:   "different.service",
	}

	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)
	url.SubURL = common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithPath("test.service"),
	)

	assert.False(t, isRouterMatch(routerCfg, url, ""))
}

// TestIsRouterMatch_ApplicationMatch tests application router matching
func TestIsRouterMatch_ApplicationMatch(t *testing.T) {
	routerCfg := &global.RouterConfig{
		Scope: "application",
		Key:   "test-app",
	}

	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)
	url.SetParam(constant.ApplicationKey, "test-app")

	assert.True(t, isRouterMatch(routerCfg, url, "test-app"))
}

// TestIsRouterMatch_ApplicationMismatch tests application router mismatch
func TestIsRouterMatch_ApplicationMismatch(t *testing.T) {
	routerCfg := &global.RouterConfig{
		Scope: "application",
		Key:   "different-app",
	}

	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)
	url.SetParam(constant.ApplicationKey, "test-app")

	assert.False(t, isRouterMatch(routerCfg, url, "test-app"))
}

// TestGetApplicationName_FromURL tests getting application name from URL
func TestGetApplicationName_FromURL(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)
	url.SetParam(constant.ApplicationKey, "my-app")

	assert.Equal(t, "my-app", getApplicationName(url))
}

// TestGetApplicationName_FromSubURL tests getting application name from SubURL
func TestGetApplicationName_FromSubURL(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithProtocol("consumer"),
		common.WithPath("test.service"),
	)
	url.SubURL = common.NewURLWithOptions(
		common.WithProtocol("dubbo"),
		common.WithPath("test.service"),
	)
	url.SubURL.SetParam(constant.ApplicationKey, "my-app")

	assert.Equal(t, "my-app", getApplicationName(url))
}
