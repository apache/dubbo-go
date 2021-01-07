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

package conncheck

import (
	"fmt"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

const (
	connCheckDubbo1010IP    = "192.168.10.10"
	connCheckDubboUrlFormat = "dubbo://%s:20000/com.ikurento.user.UserProvider"
)

func TestDefaultConnCheckerIsHealthy(t *testing.T) {
	defer protocol.CleanAllStatus()
	url, _ := common.NewURL(fmt.Sprintf(connCheckDubboUrlFormat, connCheckDubbo1010IP))
	cc := NewDefaultConnChecker(url).(*DefaultConnChecker)
	invoker := NewMockInvoker(url)
	healthy := cc.IsConnHealthy(invoker)
	assert.True(t, healthy)

	invoker = NewMockInvoker(url)
	cc = NewDefaultConnChecker(url).(*DefaultConnChecker)
	// add to black list
	protocol.SetInvokerUnhealthyStatus(invoker)
	assert.False(t, cc.IsConnHealthy(invoker))
}
