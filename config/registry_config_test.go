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
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
)

func Test_loadRegistries(t *testing.T) {
	target := "shanghai1"
	regs := map[string]*RegistryConfig{

		"shanghai1": {
			Protocol:   "mock",
			TimeoutStr: "2s",
			Group:      "shanghai_idc",
			Address:    "127.0.0.2:2181,128.0.0.1:2181",
			Username:   "user1",
			Password:   "pwd1",
		},
	}
	urls := loadRegistries(target, regs, common.CONSUMER)
	fmt.Println(urls[0])
	assert.Equal(t, "127.0.0.2:2181,128.0.0.1:2181", urls[0].Location)
}
func Test_loadRegistries1(t *testing.T) {
	target := "shanghai1"
	regs := map[string]*RegistryConfig{

		"shanghai1": {
			Protocol:   "mock",
			TimeoutStr: "2s",
			Group:      "shanghai_idc",
			Address:    "127.0.0.2:2181",
			Username:   "user1",
			Password:   "pwd1",
		},
	}
	urls := loadRegistries(target, regs, common.CONSUMER)
	fmt.Println(urls[0])
	assert.Equal(t, "127.0.0.2:2181", urls[0].Location)
}
