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
	"github.com/dubbo/dubbo-go/registry"
)

func testDubborpc(clientConfig *examples.ClientConfig, userKey string) {
	var (
		err        error
		svc        string
		method     string
		serviceIdx int
		user       *DubboUser
		conf       registry.DefaultServiceConfig
	)
	serviceIdx = -1
	svc = "com.ikurento.user.UserProvider"
	for i := range clientConfig.Service_List {
		if clientConfig.Service_List[i].Service == svc && clientConfig.Service_List[i].Protocol == public.CODECTYPE_DUBBO.String() {
			serviceIdx = i
			break
		}
	}
	if serviceIdx == -1 {
		panic(fmt.Sprintf("can not find service in config service list:%#v", clientConfig.Service_List))
	}

	// Create request
	method = string("GetUser")
	conf = registry.DefaultServiceConfig{
		Group:    clientConfig.Service_List[serviceIdx].Group,
		Protocol: public.CodecType(public.CODECTYPE_DUBBO).String(),
		Version:  clientConfig.Service_List[serviceIdx].Version,
		Service:  clientConfig.Service_List[serviceIdx].Service,
	}

	// registry pojo
	hessian.RegisterJavaEnum(Gender(MAN))
	hessian.RegisterJavaEnum(Gender(WOMAN))
	hessian.RegisterPOJO(&DubboUser{})

	user = new(DubboUser)
	defer clientInvoker.DubboClient.Close()
	err = clientInvoker.DubboCall(1, conf, method, []interface{}{userKey}, user, dubbo.WithCallRequestTimeout(10e9), dubbo.WithCallResponseTimeout(10e9), dubbo.WithCallSerialID(dubbo.S_Dubbo))
	// Call service
	if err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		return
	}

	log.Info("response result:%s", user)
}
