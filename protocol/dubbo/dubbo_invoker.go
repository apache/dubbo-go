package dubbo

import (
	"context"
	"errors"
	"strconv"
	"time"
)

import (
	"github.com/dubbo/dubbo-go/common/constant"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
)

var Err_No_Reply = errors.New("no reply")

type DubboInvoker struct {
	ctx    context.Context
	url    config.URL
	client *Client
}

func NewDubboInvoker(ctx context.Context, url config.URL, client *Client) *DubboInvoker {
	return &DubboInvoker{
		ctx:    ctx,
		url:    url,
		client: client,
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
		async = true
	}
	if async {
		err = di.client.CallOneway(url.Location, *url, inv.MethodName(),
			WithCallRequestTimeout(inv.Params()["requestTimeout"].(time.Duration)),
			WithCallResponseTimeout(inv.Params()["responseTimeout"].(time.Duration)), WithCallSerialID(inv.Params()["serialID"].(SerialID)),
			WithCallMeta_All(inv.Params()["callMeta"].(map[interface{}]interface{})))
	} else {
		if inv.Reply() == nil {
			result.Err = Err_No_Reply
		}
		err = di.client.Call(url.Location, *url, inv.MethodName(), inv.Reply(),
			WithCallRequestTimeout(inv.Params()["requestTimeout"].(time.Duration)),
			WithCallResponseTimeout(inv.Params()["responseTimeout"].(time.Duration)), WithCallSerialID(inv.Params()["serialID"].(SerialID)),
			WithCallMeta_All(inv.Params()["callMeta"].(map[interface{}]interface{})))
	}

	return result
}

func (di *DubboInvoker) GetUrl() config.IURL {
	return &di.url
}

func (di *DubboInvoker) IsAvailable() bool {
	return true
}

func (di *DubboInvoker) Destroy() {
	//todo:
}
