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

package metrics

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestGetInvokerKey(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	key := getInvokerKey(url)
	assert.Equal(t, "/com.test.UserService", key)
}

func TestGetInvokerKeyWithEmptyPath(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/")
	require.NoError(t, err)
	key := getInvokerKey(url)
	assert.Equal(t, "/", key)
}

func TestGetInstanceKey(t *testing.T) {
	url, err := common.NewURL("dubbo://127.0.0.1:20000/com.test.UserService")
	require.NoError(t, err)
	key := getInstanceKey(url)
	assert.Equal(t, "127.0.0.1:20000", key)
}

func TestGetInstanceKeyWithDifferentPort(t *testing.T) {
	url, err := common.NewURL("dubbo://192.168.1.100:8080/com.test.Service")
	require.NoError(t, err)
	key := getInstanceKey(url)
	assert.Equal(t, "192.168.1.100:8080", key)
}
