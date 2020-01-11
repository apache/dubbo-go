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

const testyml = "testdata/router_config.yml"

func TestString(t *testing.T) {

	s := "a1=>a2"
	s1 := "=>a2"
	s2 := "a1=>"

	n := strings.SplitN(s, "=>", 2)
	n1 := strings.SplitN(s1, "=>", 2)
	n2 := strings.SplitN(s2, "=>", 2)

	println(n[0], " ", n[1])
	println(n1[0], " ", n1[1])
	println(n2[0], " ", n2[1])

}

func TestLoadYmlConfig(t *testing.T) {
	routerConfig = &ConditionRouterConfig{}
	e := loadYmlConfig(testyml, routerConfig)
	assert.Nil(t, e)
	assert.NotNil(t, routerConfig)
	assert.Equal(t, routerConfig.Priority, 1)
	assert.Equal(t, routerConfig.Force, true)
	assert.Equal(t, len(routerConfig.Conditions), 2)
}

func TestRouterInit(t *testing.T) {
	e := RouterInit("testdata/router_config.yml")
	assert.Nil(t, e)
	assert.NotNil(t, routerConfig)
	assert.Equal(t, routerConfig.Priority, 1)
	assert.Equal(t, routerConfig.Force, true)
	assert.Equal(t, len(routerConfig.Conditions), 2)
}

func TestParseCondition(t *testing.T) {
	s := make([]string, 2)
	s = append(s, "a => b")
	s = append(s, "c => d")
	condition := parseCondition(s)
	assert.Equal(t, "a & c => b & d", condition)
}

func TestInitRouterUrl(t *testing.T) {
	routerConfig = &ConditionRouterConfig{}
	loadYmlConfig(testyml, routerConfig)
	url := initRouterUrl()
	assert.Equal(t, url.Protocol, "route")
	assert.Equal(t, url.Ip, "0.0.0.0")
}
