package rest

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
	"time"
)

var (
	restProtocol *RestProtocol
)

const REST = "rest"

func init() {
	extension.SetProtocol(REST, GetRestProtocol)
}

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

	return nil
}

func (rp *RestProtocol) Refer(url common.URL) protocol.Invoker {
	// create rest_invoker
	var requestTimeout = config.GetConsumerConfig().RequestTimeout

	requestTimeoutStr := url.GetParam(constant.TIMEOUT_KEY, config.GetConsumerConfig().Request_Timeout)
	connectTimeout := config.GetConsumerConfig().ConnectTimeout
	if t, err := time.ParseDuration(requestTimeoutStr); err == nil {
		requestTimeout = t
	}
	restConfig := GetRestConsumerServiceConfig(url.Service())
	restClient := extension.GetRestClient(restConfig.Client, &rest_interface.RestOptions{RequestTimeout: requestTimeout, ConnectTimeout: connectTimeout})
	invoker := NewRestInvoker(url, restClient, restConfig.RestMethodConfigsMap)
	rp.SetInvokers(invoker)
	return invoker
}

func (rp *RestProtocol) Destroy() {
	// destroy rest_server
	rp.BaseProtocol.Destroy()
}

func GetRestProtocol() protocol.Protocol {
	if restProtocol == nil {
		restProtocol = NewRestProtocol()
	}
	return restProtocol
}
