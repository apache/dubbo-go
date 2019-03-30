package main

import (
	"context"
	"fmt"
	_ "net/http/pprof"
)

import (
	// "github.com/AlexStocks/goext/log"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/jsonrpc"
	"github.com/dubbo/dubbo-go/public"
	"github.com/dubbo/dubbo-go/registry"
)

func testJsonrpc(userKey string) {
	var (
		err        error
		service    string
		method     string
		serviceIdx int
		user       *JsonRPCUser
		ctx        context.Context
		conf       registry.ServiceConfig
		req        jsonrpc.Request
		serviceURL *registry.ServiceURL
		clt        *jsonrpc.HTTPClient
	)

	clt = jsonrpc.NewHTTPClient(
		&jsonrpc.HTTPOptions{
			HandshakeTimeout: clientConfig.connectTimeout,
			HTTPTimeout:      clientConfig.requestTimeout,
		},
	)

	serviceIdx = -1
	service = "com.ikurento.user.UserProvider"
	for i := range clientConfig.Service_List {
		if clientConfig.Service_List[i].Service == service && clientConfig.Service_List[i].Protocol == public.CODECTYPE_JSONRPC.String() {
			serviceIdx = i
			break
		}
	}
	if serviceIdx == -1 {
		panic(fmt.Sprintf("can not find service in config service list:%#v", clientConfig.Service_List))
	}

	// Create request
	method = string("GetUser")
	// gxlog.CInfo("jsonrpc selected service %#v", clientConfig.Service_List[serviceIdx])
	conf = registry.ServiceConfig{
		Group:    clientConfig.Service_List[serviceIdx].Group,
		Protocol: public.CodecType(public.CODECTYPE_JSONRPC).String(),
		Version:  clientConfig.Service_List[serviceIdx].Version,
		Service:  clientConfig.Service_List[serviceIdx].Service,
	}
	// Attention the last parameter : []UserKey{userKey}
	req = clt.NewRequest(conf, method, []string{userKey})

	serviceURL, err = clientRegistry.Filter(req.ServiceConfig(), 1)
	if err != nil {
		log.Error("registry.Filter(conf:%#v) = error:%s", req.ServiceConfig(), jerrors.ErrorStack(err))
		// gxlog.CError("registry.Filter(conf:%#v) = error:%s", req.ServiceConfig(), jerrors.ErrorStack(err))
		return
	}
	log.Debug("got serviceURL: %s", serviceURL)
	// Set arbitrary headers in context
	ctx = context.WithValue(context.Background(), public.DUBBOGO_CTX_KEY, map[string]string{
		"X-Proxy-Id": "dubbogo",
		"X-Services": service,
		"X-Method":   method,
	})

	user = new(JsonRPCUser)
	// Call service
	if err = clt.Call(ctx, *serviceURL, req, user); err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		// gxlog.CError("client.Call() return error:%+v", jerrors.ErrorStack(err))
		return
	}

	log.Info("response result:%s", user)
	// gxlog.CInfo("response result:%s", user)
}

func testJsonrpcIllegalMethod(userKey string) {
	var (
		err        error
		service    string
		method     string
		serviceIdx int
		user       *JsonRPCUser
		ctx        context.Context
		conf       registry.ServiceConfig
		req        jsonrpc.Request
		serviceURL *registry.ServiceURL
		clt        *jsonrpc.HTTPClient
	)

	clt = jsonrpc.NewHTTPClient(
		&jsonrpc.HTTPOptions{
			HandshakeTimeout: clientConfig.connectTimeout,
			HTTPTimeout:      clientConfig.requestTimeout,
		},
	)

	serviceIdx = -1
	service = "com.ikurento.user.UserProvider"
	for i := range clientConfig.Service_List {
		if clientConfig.Service_List[i].Service == service && clientConfig.Service_List[i].Protocol == public.CODECTYPE_JSONRPC.String() {
			serviceIdx = i
			break
		}
	}
	if serviceIdx == -1 {
		panic(fmt.Sprintf("can not find service in config service list:%#v", clientConfig.Service_List))
	}

	// Create request
	method = string("GetUser1")
	// gxlog.CInfo("jsonrpc selected service %#v", clientConfig.Service_List[serviceIdx])
	conf = registry.ServiceConfig{
		Group:    clientConfig.Service_List[serviceIdx].Group,
		Protocol: public.CodecType(public.CODECTYPE_JSONRPC).String(),
		Version:  clientConfig.Service_List[serviceIdx].Version,
		Service:  clientConfig.Service_List[serviceIdx].Service,
	}
	// Attention the last parameter : []UserKey{userKey}
	req = clt.NewRequest(conf, method, []string{userKey})

	serviceURL, err = clientRegistry.Filter(req.ServiceConfig(), 1)
	if err != nil {
		log.Error("registry.Filter(conf:%#v) = error:%s", req.ServiceConfig(), jerrors.ErrorStack(err))
		// gxlog.CError("registry.Filter(conf:%#v) = error:%s", req.ServiceConfig(), jerrors.ErrorStack(err))
		return
	}
	log.Debug("got serviceURL: %s", serviceURL)
	// Set arbitrary headers in context
	ctx = context.WithValue(context.Background(), public.DUBBOGO_CTX_KEY, map[string]string{
		"X-Proxy-Id": "dubbogo",
		"X-Services": service,
		"X-Method":   method,
	})

	user = new(JsonRPCUser)
	// Call service
	if err = clt.Call(ctx, *serviceURL, req, user); err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		// gxlog.CError("client.Call() return error:%+v", jerrors.ErrorStack(err))
		return
	}

	log.Info("response result:%s", user)
	// gxlog.CInfo("response result:%s", user)
}
