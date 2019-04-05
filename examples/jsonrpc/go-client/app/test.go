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
	"github.com/dubbo/dubbo-go/client"
	"github.com/dubbo/dubbo-go/examples"
	"github.com/dubbo/dubbo-go/public"
)

func testJsonrpc(clientConfig *examples.ClientConfig, userKey string, method string) {
	var (
		err        error
		svc        string
		serviceIdx int
		user       *JsonRPCUser
		ctx        context.Context
		req        client.Request
	)

	serviceIdx = -1
	svc = "com.ikurento.user.UserProvider"
	for i := range clientConfig.ServiceConfigList {
		if clientConfig.ServiceConfigList[i].Service() == svc && clientConfig.ServiceConfigList[i].Protocol() == public.CODECTYPE_JSONRPC.String() {
			serviceIdx = i
			break
		}
	}
	if serviceIdx == -1 {
		panic(fmt.Sprintf("can not find service in config service list:%#v", clientConfig.ServiceConfigList))
	}

	// Create request
	// gxlog.CInfo("jsonrpc selected service %#v", clientConfig.ServiceConfigList[serviceIdx])

	// Attention the last parameter : []UserKey{userKey}
	req, err = clientInvoker.HttpClient.NewRequest(clientConfig.ServiceConfigList[serviceIdx], method, []string{userKey})

	if err != nil {
		panic(err)
	}

	ctx = context.WithValue(context.Background(), public.DUBBOGO_CTX_KEY, map[string]string{
		"X-Proxy-Id": "dubbogo",
		"X-Services": svc,
		"X-Method":   method,
	})

	user = new(JsonRPCUser)

	err = clientInvoker.HttpCall(ctx, 1, req, user)
	if err != nil {
		panic(err)
	} else {
		gxlog.CInfo("response result:%s", user)
	}

}
