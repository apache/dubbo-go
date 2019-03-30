package main

import (
	"context"
	"fmt"
	_ "net/http/pprof"
)

import (
	"github.com/AlexStocks/goext/log"
)

import (
	"github.com/dubbo/dubbo-go/examples"
	"github.com/dubbo/dubbo-go/jsonrpc"
	"github.com/dubbo/dubbo-go/public"
	"github.com/dubbo/dubbo-go/service"
)

func testJsonrpc(clientConfig *examples.ClientConfig, userKey string, method string) {
	var (
		err        error
		svc        string
		serviceIdx int
		user       *JsonRPCUser
		ctx        context.Context
		conf       service.ServiceConfig
		req        jsonrpc.Request
	)

	serviceIdx = -1
	svc = "com.ikurento.user.UserProvider"
	for i := range clientConfig.Service_List {
		if clientConfig.Service_List[i].Service == svc && clientConfig.Service_List[i].Protocol == public.CODECTYPE_JSONRPC.String() {
			serviceIdx = i
			break
		}
	}
	if serviceIdx == -1 {
		panic(fmt.Sprintf("can not find service in config service list:%#v", clientConfig.Service_List))
	}

	// Create request
	// gxlog.CInfo("jsonrpc selected service %#v", clientConfig.Service_List[serviceIdx])
	conf = service.ServiceConfig{
		Group:    clientConfig.Service_List[serviceIdx].Group,
		Protocol: public.CodecType(public.CODECTYPE_JSONRPC).String(),
		Version:  clientConfig.Service_List[serviceIdx].Version,
		Service:  clientConfig.Service_List[serviceIdx].Service,
	}
	// Attention the last parameter : []UserKey{userKey}
	req = clientInvoker.HttpClient.NewRequest(conf, method, []string{userKey})

	ctx = context.WithValue(context.Background(), public.DUBBOGO_CTX_KEY, map[string]string{
		"X-Proxy-Id": "dubbogo",
		"X-Services": svc,
		"X-Method":   method,
	})

	user = new(JsonRPCUser)

	err = clientInvoker.HttpCall(ctx, 1, &conf, req, user)
	if err != nil {
		panic(err)
	} else {
		gxlog.CInfo("response result:%s", user)
	}

}
