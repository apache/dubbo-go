// Copyright 2016-2019 Yincheng Fang
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		pendingResponses: make(map[SequenceType]*PendingResponse),
		conf:             *clientConf,
	}
	c.pool = newGettyRPCClientConnPool(c, clientConf.PoolSize, time.Duration(int(time.Second)*clientConf.PoolTTL))

	invoker := NewDubboInvoker(url, c)
	user := &User{}

	inv := invocation.NewRPCInvocationForConsumer("GetUser", nil, []interface{}{"1", "username"}, user, nil, url, nil)

	// Call
	res := invoker.Invoke(inv)
	assert.NoError(t, res.Error())
	assert.Equal(t, User{Id: "1", Name: "username"}, *res.Result().(*User))

	// CallOneway
	inv.SetAttachments(constant.ASYNC_KEY, "true")
	res = invoker.Invoke(inv)
	assert.NoError(t, res.Error())

	// AsyncCall
	lock := sync.Mutex{}
	lock.Lock()
	inv.SetCallBack(func(response CallResponse) {
		assert.Equal(t, User{Id: "1", Name: "username"}, *response.Reply.(*User))
		lock.Unlock()
	})
	res = invoker.Invoke(inv)
	assert.NoError(t, res.Error())

	// Err_No_Reply
	inv.SetAttachments(constant.ASYNC_KEY, "false")
	inv.SetReply(nil)
	res = invoker.Invoke(inv)
	assert.EqualError(t, res.Error(), "request need @reply")

	// destroy
	lock.Lock()
	proto.Destroy()
	lock.Unlock()
}
