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

package tag

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common/constant"
)

const (
	fileTestTag = `priority: 100
force: true`
)

func TestNewFileTagRouter(t *testing.T) {
	router, e := NewFileTagRouter([]byte(fileTestTag))
	assert.Nil(t, e)
	assert.NotNil(t, router)
	assert.Equal(t, 100, router.routerRule.Priority)
	assert.Equal(t, true, router.routerRule.Force)
}

func TestFileTagRouterURL(t *testing.T) {
	router, e := NewFileTagRouter([]byte(fileTestTag))
	assert.Nil(t, e)
	assert.NotNil(t, router)
	url := router.URL()
	assert.NotNil(t, url)
	force := url.GetParam(constant.ForceUseTag, "false")
	priority := url.GetParam(constant.RouterPriority, "0")
	assert.Equal(t, "true", force)
	assert.Equal(t, "100", priority)

}

func TestFileTagRouterPriority(t *testing.T) {
	router, e := NewFileTagRouter([]byte(fileTestTag))
	assert.Nil(t, e)
	assert.NotNil(t, router)
	priority := router.Priority()
	assert.Equal(t, int64(100), priority)
}
