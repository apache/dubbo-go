package config_compat

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// these functions are used to resolve circular dependencies temporarily.
var NewInfoInvoker func(url *common.URL, info interface{}, svc common.RPCService) protocol.Invoker
