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

package nacos

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/config"
)

func TestNewNacosClient(t *testing.T) {
	rc := &config.RemoteConfig{}
	client, err := NewNacosClient(rc)

	// address is nil
	assert.Nil(t, client)
	assert.NotNil(t, err)

	rc.Address = "console.nacos.io:80:123"
	client, err = NewNacosClient(rc)
	// invalid address
	assert.Nil(t, client)
	assert.NotNil(t, err)

	rc.Address = "console.nacos.io:80"
	rc.TimeoutStr = "10s"
	client, err = NewNacosClient(rc)
	assert.NotNil(t, client)
	assert.Nil(t, err)
}
