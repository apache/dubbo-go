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

package config_center

import (
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestWithTimeout(t *testing.T) {
	fa := WithTimeout(12 * time.Second)
	opt := &Options{}
	fa(opt)
	assert.Equal(t, 12*time.Second, opt.Timeout)
}

func TestGetRuleKey(t *testing.T) {
	url, err := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider?interface=test&group=groupA&version=0")
	assert.NoError(t, err)
	assert.Equal(t, "test:0:groupA", GetRuleKey(url))
}
