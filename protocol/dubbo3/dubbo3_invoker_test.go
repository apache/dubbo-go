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

package dubbo3

import (
	"context"
	"reflect"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol/dubbo3/internal"
	"github.com/apache/dubbo-go/protocol/invocation"
)

func TestInvoke(t *testing.T) {
	go internal.InitDubboServer()
	time.Sleep(time.Second * 3)

	url, err := common.NewURL(mockDubbo3CommonUrl)
	assert.Nil(t, err)

	invoker, err := NewDubboInvoker(url)

	assert.Nil(t, err)

	args := []reflect.Value{}
	args = append(args, reflect.ValueOf(&internal.HelloRequest{Name: "request name"}))
	bizReply := &internal.HelloReply{}
	invo := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("SayHello"),
		invocation.WithParameterValues(args), invocation.WithReply(bizReply))
	res := invoker.Invoke(context.Background(), invo)
	assert.Nil(t, res.Error())
	assert.NotNil(t, res.Result())
	assert.Equal(t, "Hello request name", bizReply.Message)
}
