package main

import (
	"fmt"
	_ "net/http/pprof"
)

import (
	// "github.com/AlexStocks/goext/log"
	log "github.com/AlexStocks/log4go"
	"github.com/dubbogo/hessian2"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/dubbo"
	"github.com/dubbo/dubbo-go/examples"
	"github.com/dubbo/dubbo-go/public"
)

func testDubborpc(clientConfig *examples.ClientConfig, userKey string) {
	var (
		err        error
		svc        string
		method     string
		serviceIdx int
		user       *DubboUser
	)
	serviceIdx = -1
	svc = "com.ikurento.user.UserProvider"
	for i := range clientConfig.ServiceConfig_List {
		if clientConfig.ServiceConfig_List[i].Service() == svc && clientConfig.ServiceConfig_List[i].Protocol() == public.CODECTYPE_DUBBO.String() {
			serviceIdx = i
			break
		}
	}
	if serviceIdx == -1 {
		panic(fmt.Sprintf("can not find service in config service list:%#v", clientConfig.ServiceConfig_List))
	}

	// Create request
	method = string("GetUser")

	// registry pojo
	hessian.RegisterJavaEnum(Gender(MAN))
	hessian.RegisterJavaEnum(Gender(WOMAN))
	hessian.RegisterPOJO(&DubboUser{})
	hessian.RegisterPOJO(&Response{})

	user = new(DubboUser)
	defer clientInvoker.DubboClient.Close()
	err = clientInvoker.DubboCall(1, clientConfig.ServiceConfig_List[serviceIdx], method, []interface{}{userKey}, user, dubbo.WithCallRequestTimeout(10e9), dubbo.WithCallResponseTimeout(10e9), dubbo.WithCallSerialID(dubbo.S_Dubbo))
	// Call service
	if err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		return
	}

	log.Info("response result:%s", user)
}
