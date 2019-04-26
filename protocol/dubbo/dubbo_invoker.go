package dubbo

import (
	"context"
	"errors"
	"strconv"
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/dubbo-go/common/constant"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
)

var Err_No_Reply = errors.New("no reply")

type DubboInvoker struct {
	ctx         context.Context
	url         config.URL
	client      *Client
	available   bool
	destroyed   bool
	destroyLock sync.Mutex
}

func NewDubboInvoker(ctx context.Context, url config.URL, client *Client) *DubboInvoker {
	return &DubboInvoker{
		ctx:       ctx,
		url:       url,
		client:    client,
		available: true,
		destroyed: false,
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
			result.Err = di.client.AsyncCall(url.Location, *url, inv.MethodName(), inv.Reply(), callBack,
				WithCallRequestTimeout(inv.Params()["requestTimeout"].(time.Duration)),
				WithCallResponseTimeout(inv.Params()["responseTimeout"].(time.Duration)),
				WithCallSerialID(inv.Params()["serialID"].(SerialID)),
				WithCallMeta_All(inv.Params()["callMeta"].(map[interface{}]interface{})))
		} else {
			result.Err = di.client.CallOneway(url.Location, *url, inv.MethodName(),
				WithCallRequestTimeout(inv.Params()["requestTimeout"].(time.Duration)),
				WithCallResponseTimeout(inv.Params()["responseTimeout"].(time.Duration)),
				WithCallSerialID(inv.Params()["serialID"].(SerialID)),
				WithCallMeta_All(inv.Params()["callMeta"].(map[interface{}]interface{})))
		}
	} else {
		if inv.Reply() == nil {
			result.Err = Err_No_Reply
		} else {
			result.Err = di.client.Call(url.Location, *url, inv.MethodName(), inv.Reply(),
				WithCallRequestTimeout(inv.Params()["requestTimeout"].(time.Duration)),
				WithCallResponseTimeout(inv.Params()["responseTimeout"].(time.Duration)),
				WithCallSerialID(inv.Params()["serialID"].(SerialID)),
				WithCallMeta_All(inv.Params()["callMeta"].(map[interface{}]interface{})))
			result.Rest = inv.Reply() // reply should be set to result.Rest when sync
		}
	}

	return result
}

func (di *DubboInvoker) GetUrl() config.IURL {
	return &di.url
}

func (di *DubboInvoker) IsAvailable() bool {
	return di.available
}

func (di *DubboInvoker) Destroy() {
	if di.destroyed {
		return
	}
	di.destroyLock.Lock()
	defer di.destroyLock.Unlock()

	di.destroyed = true
	di.available = false

	di.client.Close() // close client
}
