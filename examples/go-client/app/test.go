package main

import (
	"context"
	"fmt"
	"github.com/dubbogo/hessian2"
	_ "net/http/pprof"
)

import (
	// "github.com/AlexStocks/goext/log"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	dubbo "github.com/dubbo/dubbo-go/dubbo"
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

func testDubborpc(userKey string) {
	var (
		err        error
		service    string
		method     string
		serviceIdx int
		user       *DubboUser
		conf       registry.ServiceConfig
		serviceURL *registry.ServiceURL
		cltD       *dubbo.Client
	)

	cltD, err = dubbo.NewClient(&dubbo.ClientConfig{
		PoolSize:        64,
		PoolTTL:         600,
		ConnectionNum:   2, // 不能太大
		FailFastTimeout: "5s",
		SessionTimeout:  "20s",
		HeartbeatPeriod: "5s",
		GettySessionParam: dubbo.GettySessionParam{
			CompressEncoding: false, // 必须false
			TcpNoDelay:       true,
			KeepAlivePeriod:  "120s",
			TcpRBufSize:      262144,
			TcpKeepAlive:     true,
			TcpWBufSize:      65536,
			PkgRQSize:        1024,
			PkgWQSize:        512,
			TcpReadTimeout:   "1s",
			TcpWriteTimeout:  "5s",
			WaitTimeout:      "1s",
			MaxMsgLen:        1024,
			SessionName:      "client",
		},
	})
	if err != nil {
		log.Error("hessian.NewClient(conf) = error:%s", jerrors.ErrorStack(err))
		return
	}
	defer cltD.Close()
	serviceIdx = -1
	service = "com.ikurento.user.UserProvider"
	for i := range clientConfig.Service_List {
		if clientConfig.Service_List[i].Service == service && clientConfig.Service_List[i].Protocol == public.CODECTYPE_DUBBO.String() {
			serviceIdx = i
			break
		}
	}
	if serviceIdx == -1 {
		panic(fmt.Sprintf("can not find service in config service list:%#v", clientConfig.Service_List))
	}

	// Create request
	method = string("GetUser")
	conf = registry.ServiceConfig{
		Group:    clientConfig.Service_List[serviceIdx].Group,
		Protocol: public.CodecType(public.CODECTYPE_DUBBO).String(),
		Version:  clientConfig.Service_List[serviceIdx].Version,
		Service:  clientConfig.Service_List[serviceIdx].Service,
	}

	serviceURL, err = clientRegistry.Filter(conf, 1)
	if err != nil {
		log.Error("registry.Filter(conf:%#v) = error:%s", conf, jerrors.ErrorStack(err))
		return
	}
	log.Debug("got serviceURL: %s", serviceURL)

	// registry pojo
	hessian.RegisterJavaEnum(Gender(MAN))
	hessian.RegisterJavaEnum(Gender(WOMAN))
	hessian.RegisterPOJO(&DubboUser{})
	hessian.RegisterPOJO(&Response{})

	user = new(DubboUser)
	err = cltD.Call(serviceURL.Ip+":"+serviceURL.Port, *serviceURL, method, []interface{}{userKey}, user, dubbo.WithCallRequestTimeout(10e9), dubbo.WithCallResponseTimeout(10e9), dubbo.WithCallSerialID(dubbo.S_Default))
	// Call service
	if err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		return
	}

	log.Info("response result:%s", user)
}
