package dubbo

import (
	"errors"
	"strconv"
	"sync"
)

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/dubbo-go/common/constant"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
)

var Err_No_Reply = errors.New("request need @reply")

type DubboInvoker struct {
	protocol.BaseInvoker
	client      *Client
	destroyLock sync.Mutex
}

func NewDubboInvoker(url config.IURL, client *Client) *DubboInvoker {
	return &DubboInvoker{
		BaseInvoker: *protocol.NewBaseInvoker(url),
		client:      client,
	}
}

func (di *DubboInvoker) Invoke(invocation protocol.Invocation) protocol.Result {

	var (
		err    error
		result protocol.RPCResult
	)

	inv := invocation.(*protocol.RPCInvocation)
	url := inv.Invoker().GetUrl().(*config.URL)
	// async
	async, err := strconv.ParseBool(inv.AttachmentsByKey(constant.ASYNC_KEY, "false"))
	if err != nil {
		log.Error("ParseBool - error: %v", err)
		async = false
	}
	if async {
		if callBack, ok := inv.CallBack().(func(response CallResponse)); ok {
			result.Err = di.client.AsyncCall(url.Location, *url, inv.MethodName(), inv.Arguments(), callBack, inv.Reply())
		} else {
			result.Err = di.client.CallOneway(url.Location, *url, inv.MethodName(), inv.Arguments())
		}
	} else {
		if inv.Reply() == nil {
			result.Err = Err_No_Reply
		} else {
			result.Err = di.client.Call(url.Location, *url, inv.MethodName(), inv.Arguments(), inv.Reply())
			result.Rest = inv.Reply() // reply should be set to result.Rest when sync
		}
	}

	return &result
}

func (di *DubboInvoker) Destroy() {
	if di.IsDestroyed() {
		return
	}
	di.destroyLock.Lock()
	defer di.destroyLock.Unlock()

	if di.IsDestroyed() {
		return
	}

	di.BaseInvoker.Destroy()

	if di.client != nil {
		di.client.Close() // close client
	}
}
