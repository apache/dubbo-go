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
package registry

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestKey(t *testing.T) {
	u1, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.0")
	se := ServiceEvent{
		Service: u1,
	}
	assert.Equal(t, se.Key(), "dubbo://:@127.0.0.1:20000/?interface=com.ikurento.user.UserProvider&group=&version=2.0&timestamp=")

	se2 := ServiceEvent{
		Service: u1,
		KeyFunc: defineKey,
	}
	assert.Equal(t, se2.Key(), "Hello Key")
}

func defineKey(url *common.URL) string {
	return "Hello Key"
}
