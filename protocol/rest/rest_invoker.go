package rest

import (
	"github.com/apache/dubbo-go/protocol"
	"net/http"
)

type RestInvoker struct {
	protocol.BaseInvoker
	client *http.Client
}

func NewRestInvoker() *RestInvoker {
	return &RestInvoker{}
}

func (ri *RestInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	// TODO 首先，将本地调用和ReferRestConfig，结合在一起，到这一步完成了Service -> Rest的绑定；
	//      第一步完成的产物是一份metadata和请求参数，本步骤将利用metadata和请求参数来构造http请求，包括参数序列化，header设定；其中参数序列化可以是Json，也可以是XML，后续可以扩展到多媒体等；
	//      http client发送请求。该步骤要允许用户自定义其连接参数，例如超时时间等；
	//var (
	//	result protocol.RPCResult
	//)
	//req, err := http.NewRequest("method", "url", bytes.NewBuffer([]byte{}))
	//resp, err := ri.client.Do(req)
	//defer resp.Body.Close()
	//body, err := ioutil.ReadAll(resp.Body)
	//result.Rest = invocation.Reply()
	//return &result
	return nil
}
