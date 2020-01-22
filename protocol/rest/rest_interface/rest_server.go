package rest_interface

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

type RestServer interface {
	Start(url common.URL)
	Deploy(invoker protocol.Invoker, restMethodConfig map[string]*RestMethodConfig)
	Undeploy(restMethodConfig map[string]*RestMethodConfig)
	Destory()
}
