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
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	invocation_impl "github.com/dubbo/go-for-apache-dubbo/protocol/invocation"
)

var Err_No_Reply = errors.New("request need @reply")

type DubboInvoker struct {
	protocol.BaseInvoker
	client      *Client
	destroyLock sync.Mutex
}

func NewDubboInvoker(url common.URL, client *Client) *DubboInvoker {
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

	inv := invocation.(*invocation_impl.RPCInvocation)
	url := di.GetUrl()
	// async
	async, err := strconv.ParseBool(inv.AttachmentsByKey(constant.ASYNC_KEY, "false"))
	if err != nil {
		log.Error("ParseBool - error: %v", err)
		async = false
	}
	if async {
		if callBack, ok := inv.CallBack().(func(response CallResponse)); ok {
			result.Err = di.client.AsyncCall(url.Location, url, inv.MethodName(), inv.Arguments(), callBack, inv.Reply())
		} else {
			result.Err = di.client.CallOneway(url.Location, url, inv.MethodName(), inv.Arguments())
		}
		log.Debug("result.Err: %v, result.Rest: %v", result.Err, result.Rest)
	} else {
		if inv.Reply() == nil {
			result.Err = Err_No_Reply
		} else {

			result.Err = di.client.Call(url.Location, url, inv.MethodName(), inv.Arguments(), inv.Reply())
			result.Rest = inv.Reply() // reply should be set to result.Rest when sync
		}
		log.Debug("result.Err: %v, result.Rest: %v", result.Err, result.Rest)
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
