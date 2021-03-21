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
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
)

const (
	factoryLocalIP = "127.0.0.1"
	factoryFormat  = "dubbo://%s:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.6.0&enabled=true"
)

func TestTagRouterFactoryNewRouter(t *testing.T) {
	u1, err := common.NewURL(fmt.Sprintf(factoryFormat, factoryLocalIP))
	assert.Nil(t, err)
	factory := NewTagRouterFactory()
	notify := make(chan struct{})
	go func() {
		for range notify {
		}
	}()
	tagRouter, e := factory.NewPriorityRouter(u1, notify)
	assert.Nil(t, e)
	assert.NotNil(t, tagRouter)
}
