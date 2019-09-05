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

package dubbo

import (
	"sync"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol/invocation"
)

func TestDubboInvoker_Invoke(t *testing.T) {
	proto, url := InitTest(t)

	c := &Client{
		pendingResponses: new(sync.Map),
		conf:             *clientConf,
		opts: Options{
			ConnectTimeout: 3e9,
			RequestTimeout: 6e9,
		},
	}
	c.pool = newGettyRPCClientConnPool(c, clientConf.PoolSize, time.Duration(int(time.Second)*clientConf.PoolTTL))

	invoker := NewDubboInvoker(url, c)
	user := &User{}

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("GetUser"), invocation.WithArguments([]interface{}{"1", "username"}),
		invocation.WithReply(user), invocation.WithAttachments(map[string]string{"test_key": "test_value"}))

	// Call
	res := invoker.Invoke(inv)
	assert.NoError(t, res.Error())
	assert.Equal(t, User{Id: "1", Name: "username"}, *res.Result().(*User))
	assert.Equal(t, "test_value", res.Attachments()["test_key"]) // test attachments for request/response

	// CallOneway
	inv.SetAttachments(constant.ASYNC_KEY, "true")
	res = invoker.Invoke(inv)
	assert.NoError(t, res.Error())

	// AsyncCall
	lock := sync.Mutex{}
	lock.Lock()
	inv.SetCallBack(func(response CallResponse) {
		assert.Equal(t, User{Id: "1", Name: "username"}, *response.Reply.(*Response).reply.(*User))
		lock.Unlock()
	})
	res = invoker.Invoke(inv)
	assert.NoError(t, res.Error())

	// Err_No_Reply
	inv.SetAttachments(constant.ASYNC_KEY, "false")
	inv.SetReply(nil)
	res = invoker.Invoke(inv)
	assert.EqualError(t, res.Error(), "request need @response")

	// destroy
	lock.Lock()
	proto.Destroy()
	lock.Unlock()
}
