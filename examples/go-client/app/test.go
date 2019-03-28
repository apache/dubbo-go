package main

import (
	"context"
	"fmt"
	"github.com/AlexStocks/goext/log"
	"github.com/dubbo/dubbo-go/client/invoker"
	"github.com/dubbo/dubbo-go/client/loadBalance"
	"github.com/dubbo/dubbo-go/registry/zookeeper"
	"github.com/dubbo/dubbo-go/service"
	_ "net/http/pprof"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/jsonrpc"
	"github.com/dubbo/dubbo-go/public"
)

func testJsonrpc(userKey string,method string) {
	var (
		err        error
		svc    string
		serviceIdx int
		user       *JsonRPCUser
		ctx        context.Context
		conf       service.ServiceConfig
		req        jsonrpc.Request
		clt        *jsonrpc.HTTPClient
	)

	clt = jsonrpc.NewHTTPClient(
		&jsonrpc.HTTPOptions{
			HandshakeTimeout: clientConfig.connectTimeout,
			HTTPTimeout:      clientConfig.requestTimeout,
		},
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
	conf =  service.ServiceConfig{
			Group:    clientConfig.Service_List[serviceIdx].Group,
			Protocol: public.CodecType(public.CODECTYPE_JSONRPC).String(),
			Version:  clientConfig.Service_List[serviceIdx].Version,
			Service:  clientConfig.Service_List[serviceIdx].Service,
	}
	// Attention the last parameter : []UserKey{userKey}
	req = clt.NewRequest(conf, method, []string{userKey})

	clientRegistry = clientRegistry.(*zookeeper.ZkRegistry)

	ctx = context.WithValue(context.Background(), public.DUBBOGO_CTX_KEY, map[string]string{
		"X-Proxy-Id": "dubbogo",
		"X-Services": svc,
		"X-Method":   method,
	})

	user = new(JsonRPCUser)
	// new invoker to Call service

	invoker := invoker.NewInvoker(clientRegistry,clt,
		invoker.WithContext(ctx),
		invoker.WithLBSelector(loadBalance.NewRandomSelector()))
	err = invoker.Call(1,&conf,req,user)
	if err !=nil{
		jerrors.Errorf("call service err , msg is :",err)
	}
	 gxlog.CInfo("response result:%s", user)
}
