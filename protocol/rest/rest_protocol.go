package rest

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

var (
	restProtocol *RestProtocol
)

type RestProtocol struct {
	protocol.BaseProtocol
}

func NewRestProtocol() *RestProtocol {
	return &RestProtocol{}
}

func (rp *RestProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	// TODO 当用户注册一个服务的时候，根据ExporterConfig和服务实现，完成Service -> Rest的绑定。注意此处是Service -> Rest，因为此时我们是暴露服务。当收到请求的时候，恰好是暴露服务的反向，即Rest -> Service；
	//      Server在Export的时候并不做什么事情。但是在接受到请求的时候，它需要负责执行反序列化的过程;
	//      http server是一个抽象隔离层。它内部允许使用beego或者gin来作为web服务器，接收请求，用户可以扩展自己的实现；

	// create gin_server
	// save gin_server in map
	return nil
}

func (rp *RestProtocol) Refer(url common.URL) protocol.Invoker {
	// create rest_invoker
	return nil
}

func (rp *RestProtocol) Destroy() {
	// destroy rest_server
}

func GetRestProtocol() protocol.Protocol {
	if restProtocol == nil {
		restProtocol = NewRestProtocol()
	}
	return restProtocol
}
