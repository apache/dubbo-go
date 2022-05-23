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

package config

import (
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

const (
	testDestinationRuleYML      = "testdata/router_config_dest_rule.yml"
	errorTestDestinationRuleYML = "testdata/router_config_destination_rule_error.yml"
	testVirtualServiceYML       = "testdata/router_config_virtual_service.yml"
)

func TestString(t *testing.T) {
	s := "a1=>a2"
	s1 := "=>a2"
	s2 := "a1=>"

	n := strings.SplitN(s, "=>", 2)
	n1 := strings.SplitN(s1, "=>", 2)
	n2 := strings.SplitN(s2, "=>", 2)

	assert.Equal(t, n[0], "a1")
	assert.Equal(t, n[1], "a2")

	assert.Equal(t, n1[0], "")
	assert.Equal(t, n1[1], "a2")

	assert.Equal(t, n2[0], "a1")
	assert.Equal(t, n2[1], "")
}

func TestNewRouterConfigBuilder(t *testing.T) {
	tag := Tag{
		Name:      "tag",
		Addresses: []string{"127.0.0.1"},
	}
	rc := newEmptyRootConfig()
	config := NewRouterConfigBuilder().
		SetScope("scope").
		SetKey("key").
		SetForce(true).
		SetRuntime(true).
		SetEnabled(true).
		SetValid(true).
		SetPriority(10).
		SetConditions([]string{"condition"}).
		AddCondition("condition").
		SetTags([]Tag{tag}).
		AddTag(tag).
		Build()

	err := initRouterConfig(rc)
	assert.NoError(t, err)
	err = config.Init()
	assert.NoError(t, err)

	assert.Equal(t, config.Prefix(), constant.RouterConfigPrefix)
}
