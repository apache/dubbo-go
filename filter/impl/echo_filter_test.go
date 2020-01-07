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

package impl

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

func TestEchoFilter_Invoke(t *testing.T) {
	filter := GetFilter()
	result := filter.Invoke(protocol.NewBaseInvoker(common.URL{}),
		invocation.NewRPCInvocation("$echo", []interface{}{"OK"}, nil))
	assert.Equal(t, "OK", result.Result())

	result = filter.Invoke(protocol.NewBaseInvoker(common.URL{}),
		invocation.NewRPCInvocation("MethodName", []interface{}{"OK"}, nil))
	assert.Nil(t, result.Error())
	assert.Nil(t, result.Result())
}
