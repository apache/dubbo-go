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

package condition

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestLoadYmlConfig(t *testing.T) {
	router, e := NewFileConditionRouter([]byte(`scope: application
key: mock-app
priority: 1
force: true
conditions :
  - "a => b"
  - "c => d"`))
	assert.Nil(t, e)
	assert.NotNil(t, router)
	assert.Equal(t, router.routerRule.Priority, 1)
	assert.Equal(t, router.routerRule.Force, true)
	assert.Equal(t, len(router.routerRule.Conditions), 2)
}

func TestParseCondition(t *testing.T) {
	s := make([]string, 2)
	s = append(s, "a => b")
	s = append(s, "c => d")
	condition := parseCondition(s)
	assert.Equal(t, "a & c => b & d", condition)
}

func TestFileRouterURL(t *testing.T) {
	router, e := NewFileConditionRouter([]byte(`scope: application
key: mock-app
priority: 1
force: true
conditions :
  - "a => b"
  - "c => d"`))
	assert.Nil(t, e)
	assert.NotNil(t, router)
	assert.Equal(t, "condition://0.0.0.0:?application=mock-app&category=routers&force=true&priority=1&router=condition&rule=YSAmIGMgPT4gYiAmIGQ%3D", router.URL().String())

	router, e = NewFileConditionRouter([]byte(`scope: service
key: mock-service
priority: 1
force: true
conditions :
  - "a => b"
  - "c => d"`))
	assert.Nil(t, e)
	assert.NotNil(t, router)
	assert.Equal(t, "condition://0.0.0.0:?category=routers&force=true&interface=mock-service&priority=1&router=condition&rule=YSAmIGMgPT4gYiAmIGQ%3D", router.URL().String())

	router, e = NewFileConditionRouter([]byte(`scope: service
key: grp1/mock-service:v1
priority: 1
force: true
conditions :
  - "a => b"
  - "c => d"`))
	assert.Nil(t, e)
	assert.NotNil(t, router)
	assert.Equal(t, "condition://0.0.0.0:?category=routers&force=true&group=grp1&interface=mock-service&priority=1&router=condition&rule=YSAmIGMgPT4gYiAmIGQ%3D&version=v1", router.URL().String())
}

func TestParseServiceRouterKey(t *testing.T) {
	testString := " mock-group / mock-service:1.0.0"
	grp, srv, ver, err := parseServiceRouterKey(testString)
	assert.Equal(t, "mock-group", grp)
	assert.Equal(t, "mock-service", srv)
	assert.Equal(t, "1.0.0", ver)
	assert.Nil(t, err)

	testString = "mock-group/mock-service"
	grp, srv, ver, err = parseServiceRouterKey(testString)
	assert.Equal(t, "mock-group", grp)
	assert.Equal(t, "mock-service", srv)
	assert.Equal(t, "", ver)
	assert.Nil(t, err)

	testString = "mock-service:1.0.0"
	grp, srv, ver, err = parseServiceRouterKey(testString)
	assert.Equal(t, "", grp)
	assert.Equal(t, "mock-service", srv)
	assert.Equal(t, "1.0.0", ver)
	assert.Nil(t, err)

	testString = "mock-service"
	grp, srv, ver, err = parseServiceRouterKey(testString)
	assert.Equal(t, "", grp)
	assert.Equal(t, "mock-service", srv)
	assert.Equal(t, "", ver)
	assert.NoError(t, err)

	testString = "/mock-service:"
	grp, srv, ver, err = parseServiceRouterKey(testString)
	assert.Equal(t, "", grp)
	assert.Equal(t, "mock-service", srv)
	assert.Equal(t, "", ver)
	assert.NoError(t, err)

	testString = "grp:mock-service:123"
	grp, srv, ver, err = parseServiceRouterKey(testString)
	assert.Equal(t, "", grp)
	assert.Equal(t, "", srv)
	assert.Equal(t, "", ver)
	assert.Error(t, err)

	testString = ""
	grp, srv, ver, err = parseServiceRouterKey(testString)
	assert.Equal(t, "", grp)
	assert.Equal(t, "", srv)
	assert.Equal(t, "", ver)
	assert.NoError(t, err)
}
