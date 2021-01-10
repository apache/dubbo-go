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

import (
	"github.com/apache/dubbo-go/common"
)

func TestGetRule(t *testing.T) {
	testyml := `
scope: application
key: test-provider
runtime: true
force: false
conditions:
  - >
    method!=sayHello =>
  - >
    ip=127.0.0.1
    =>
    1.1.1.1`
	rule, e := getRule(testyml)

	assert.Nil(t, e)
	assert.NotNil(t, rule)
	assert.Equal(t, 2, len(rule.Conditions))
	assert.Equal(t, "application", rule.Scope)
	assert.True(t, rule.Runtime)
	assert.Equal(t, false, rule.Force)
	assert.Equal(t, testyml, rule.RawRule)
	assert.True(t, rule.Valid)
	assert.Equal(t, false, rule.Enabled)
	assert.Equal(t, false, rule.Dynamic)
	assert.Equal(t, "test-provider", rule.Key)

	testyml = `
key: test-provider
runtime: true
force: false
conditions:
  - >
    method!=sayHello =>`
	rule, e = getRule(testyml)
	assert.Nil(t, e)
	assert.False(t, rule.Valid)

	testyml = `
scope: noApplication
key: test-provider
conditions:
  - >
    method!=sayHello =>`
	rule, e = getRule(testyml)
	assert.Nil(t, e)
	assert.False(t, rule.Valid)
}

func TestIsMatchGlobPattern(t *testing.T) {
	url, _ := common.NewURL("dubbo://localhost:8080/Foo?key=v*e")
	assert.Equal(t, true, isMatchGlobalPattern("$key", "value", url))
}
