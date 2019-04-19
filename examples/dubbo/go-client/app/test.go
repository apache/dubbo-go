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
	for i := range clientConfig.ServiceConfigList {
		if clientConfig.ServiceConfigList[i].Service() == svc && clientConfig.ServiceConfigList[i].Protocol() == public.CODECTYPE_DUBBO.String() {
			serviceIdx = i
			break
		}
	}
	if serviceIdx == -1 {
		panic(fmt.Sprintf("can not find service in config service list:%#v", clientConfig.ServiceConfigList))
	}

	// Create request
	method = string("GetUser")

	// registry pojo
	hessian.RegisterJavaEnum(Gender(MAN))
	hessian.RegisterJavaEnum(Gender(WOMAN))
	hessian.RegisterPOJO(&DubboUser{})

	user = new(DubboUser)
	defer clientInvoker.DubboClient.Close()
	err = clientInvoker.DubboCall(1, clientConfig.ServiceConfigList[serviceIdx], method, []interface{}{userKey}, user, dubbo.WithCallRequestTimeout(10e9), dubbo.WithCallResponseTimeout(10e9), dubbo.WithCallSerialID(dubbo.S_Dubbo))
	// Call service
	if err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		return
	}

	log.Info("response result of method [GetUser]: %s", user)

	userProvider := new(UserProvider)
	userProviderNoErr := new(UserProviderNoErr)
	userProviderRetPtr := new(UserProviderRetPtr)
	err = clientInvoker.ImplementService(userProvider, clientConfig.ServiceConfigList[serviceIdx], dubbo.WithCallRequestTimeout(10e9), dubbo.WithCallResponseTimeout(10e9), dubbo.WithCallSerialID(dubbo.S_Dubbo))
	if err != nil {
		log.Error("ImplementService return error:%+v", jerrors.ErrorStack(err))
	}

	err = clientInvoker.ImplementService(userProviderNoErr, clientConfig.ServiceConfigList[serviceIdx], dubbo.WithCallRequestTimeout(10e9), dubbo.WithCallResponseTimeout(10e9), dubbo.WithCallSerialID(dubbo.S_Dubbo))
	if err != nil {
		panic(fmt.Sprintf("ImplementService return error: %+v", jerrors.ErrorStack(err)))
	}

	err = clientInvoker.ImplementService(userProviderRetPtr, clientConfig.ServiceConfigList[serviceIdx], dubbo.WithCallRequestTimeout(10e9), dubbo.WithCallResponseTimeout(10e9), dubbo.WithCallSerialID(dubbo.S_Dubbo))
	if err != nil {
		panic(fmt.Sprintf("ImplementService return error: %+v", jerrors.ErrorStack(err)))
	}

	dubboUser, err := userProvider.GetUser(userKey)
	if err != nil {
		log.Error("userProvider.GetUser return error:%+v", jerrors.ErrorStack(err))
	}

	log.Info("response result of method [GetUser] by ImplementService: %+v", dubboUser)

	// using userProviderNoErr will panic if has error when invoke
	dubboUserNoErr := userProviderNoErr.GetUser(userKey)
	log.Info("response result of method [GetUser] by ImplementService userProviderNoErr: %+v", dubboUserNoErr)

	dubboUserPtr, err := userProviderRetPtr.GetUser(userKey)
	switch {
	case err != nil:
		log.Error("userProviderRetPtr.GetUser return error:%+v", jerrors.ErrorStack(err))
	case dubboUserPtr == nil:
		log.Error("dubboUserPtr may not exist.")
	default:
		log.Info("response result of method [GetUser] by ImplementService userProviderRetPtr: %+v", *dubboUserPtr)
	}
}
