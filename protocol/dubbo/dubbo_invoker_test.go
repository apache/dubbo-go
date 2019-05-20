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
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/protocol/invocation"
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
