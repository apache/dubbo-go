/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package dubbo3

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/dubbogo/grpc-go"
	"github.com/dubbogo/grpc-go/metadata"

	tripleConstant "github.com/dubbogo/triple/pkg/common/constant"
	triConfig "github.com/dubbogo/triple/pkg/config"
	"github.com/dubbogo/triple/pkg/triple"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

var protocolOnce sync.Once

func init() {
	extension.SetProtocol(tripleConstant.TRIPLE, GetProtocol)
	protocolOnce = sync.Once{}
}

var (
	dubboProtocol *DubboProtocol
)

// DubboProtocol supports dubbo 3.0 protocol. It implements Protocol interface for dubbo protocol.
type DubboProtocol struct {
	protocol.BaseProtocol
	serverLock sync.Mutex
	serviceMap *sync.Map                       // serviceMap is used to export multiple service by one server
	serverMap  map[string]*triple.TripleServer // serverMap stores all exported server
}

// NewDubboProtocol create a dubbo protocol.
func NewDubboProtocol() *DubboProtocol {
	return &DubboProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*triple.TripleServer),
		serviceMap:   &sync.Map{},
	}
}

// Export export dubbo3 service.
func (dp *DubboProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetURL()
	serviceKey := url.ServiceKey()
	exporter := NewDubboExporter(serviceKey, invoker, dp.ExporterMap(), dp.serviceMap)
	dp.SetExporterMap(serviceKey, exporter)
	logger.Infof("[Triple Protocol] Export service: %s", url.String())

	key := url.GetParam(constant.BeanNameKey, "")
	var service interface{}
	service = config.GetProviderService(key)

	serializationType := url.GetParam(constant.SerializationKey, constant.ProtobufSerialization)
	var triSerializationType tripleConstant.CodecType

	if serializationType == constant.ProtobufSerialization {
		m, ok := reflect.TypeOf(service).MethodByName("XXX_SetProxyImpl")
		if !ok {
			logger.Errorf("PB service with key = %s is not support XXX_SetProxyImpl to pb."+
				"Please run go install github.com/dubbogo/tools/cmd/protoc-gen-go-triple@latest to update your "+
				"protoc-gen-go-triple and re-generate your pb file again.", key)
			return nil
		}
		if invoker == nil {
			panic(fmt.Sprintf("no invoker found for servicekey: %v", url.ServiceKey()))
		}
		in := []reflect.Value{reflect.ValueOf(service)}
		in = append(in, reflect.ValueOf(invoker))
		m.Func.Call(in)
		triSerializationType = tripleConstant.PBCodecName
	} else {
		valueOf := reflect.ValueOf(service)
		typeOf := valueOf.Type()
		numField := valueOf.NumMethod()
		tripleService := &UnaryService{proxyImpl: invoker}
		for i := 0; i < numField; i++ {
			ft := typeOf.Method(i)
			if ft.Name == "Reference" {
				continue
			}
			// get all method params type
			typs := make([]reflect.Type, 0)
			for j := 2; j < ft.Type.NumIn(); j++ {
				typs = append(typs, ft.Type.In(j))
			}
			tripleService.setReqParamsTypes(ft.Name, typs)
		}
		service = tripleService
		triSerializationType = tripleConstant.CodecType(serializationType)
	}

	dp.serviceMap.Store(url.GetParam(constant.InterfaceKey, ""), service)

	// try start server
	dp.openServer(url, triSerializationType)
	return exporter
}

// Refer create dubbo3 service reference.
func (dp *DubboProtocol) Refer(url *common.URL) protocol.Invoker {
	invoker, err := NewDubboInvoker(url)
	if err != nil {
		logger.Errorf("Refer url = %+v, with error = %s", url, err.Error())
		return nil
	}
	dp.SetInvokers(invoker)
	logger.Infof("[Triple Protocol] Refer service: %s", url.String())
	return invoker
}

// Destroy destroy dubbo3 service.
func (dp *DubboProtocol) Destroy() {
	dp.BaseProtocol.Destroy()
	keyList := make([]string, 16)

	dp.serverLock.Lock()
	defer dp.serverLock.Unlock()
	// Stop all server
	for k, _ := range dp.serverMap {
		keyList = append(keyList, k)
	}
	for _, v := range keyList {
		if server := dp.serverMap[v]; server != nil {
			server.Stop()
		}
		delete(dp.serverMap, v)
	}
}

// Dubbo3GrpcService is gRPC service
type Dubbo3GrpcService interface {
	// SetProxyImpl sets proxy.
	XXX_SetProxyImpl(impl protocol.Invoker)
	// GetProxyImpl gets proxy.
	XXX_GetProxyImpl() protocol.Invoker
	// ServiceDesc gets an RPC service's specification.
	XXX_ServiceDesc() *grpc.ServiceDesc
}

type UnaryService struct {
	proxyImpl  protocol.Invoker
	reqTypeMap sync.Map
}

func (d *UnaryService) setReqParamsTypes(methodName string, typ []reflect.Type) {
	d.reqTypeMap.Store(methodName, typ)
}

func (d *UnaryService) GetReqParamsInterfaces(methodName string) ([]interface{}, bool) {
	val, ok := d.reqTypeMap.Load(methodName)
	if !ok {
		return nil, false
	}
	typs := val.([]reflect.Type)
	reqParamsInterfaces := make([]interface{}, 0, len(typs))
	for _, typ := range typs {
		reqParamsInterfaces = append(reqParamsInterfaces, reflect.New(typ).Interface())
	}
	return reqParamsInterfaces, true
}

func (d *UnaryService) InvokeWithArgs(ctx context.Context, methodName string, arguments []interface{}) (interface{}, error) {
	dubboAttachment := make(map[string]interface{})
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		for k := range md {
			dubboAttachment[k] = md.Get(k)[0]
		}
	}
	res := d.proxyImpl.Invoke(ctx, invocation.NewRPCInvocation(methodName, arguments, dubboAttachment))
	return res, res.Error()
}

// openServer open a dubbo3 server, if there is already a service using the same protocol, it returns directly.
func (dp *DubboProtocol) openServer(url *common.URL, tripleCodecType tripleConstant.CodecType) {
	dp.serverLock.Lock()
	defer dp.serverLock.Unlock()
	_, ok := dp.serverMap[url.Location]

	if ok {
		dp.serverMap[url.Location].RefreshService()
		return
	}

	opts := []triConfig.OptionFunction{
		triConfig.WithCodecType(tripleCodecType),
		triConfig.WithLocation(url.Location),
		triConfig.WithLogger(logger.GetLogger()),
	}
	tracingKey := url.GetParam(constant.TracingConfigKey, "")
	if tracingKey != "" {
		tracingConfig := config.GetTracingConfig(tracingKey)
		if tracingConfig != nil {
			if tracingConfig.ServiceName == "" {
				tracingConfig.ServiceName = config.GetApplicationConfig().Name
			}
			switch tracingConfig.Name {
			case "jaeger":
				opts = append(opts, triConfig.WithJaegerConfig(
					tracingConfig.Address,
					tracingConfig.ServiceName,
					tracingConfig.UseAgent,
				))
			default:
				logger.Warnf("unsupported tracing name %s, now triple only support jaeger", tracingConfig.Name)
			}
		}
	}

	if maxCall := url.GetParam(constant.MaxServerRecvMsgSize, ""); maxCall != "" {
		if size, err := strconv.Atoi(maxCall); err == nil && size != 0 {
			opts = append(opts, triConfig.WithGRPCMaxServerRecvMessageSize(size))
		}
	}
	if maxCall := url.GetParam(constant.MaxServerSendMsgSize, ""); maxCall != "" {
		if size, err := strconv.Atoi(maxCall); err == nil && size != 0 {
			opts = append(opts, triConfig.WithGRPCMaxServerSendMessageSize(size))
		}
	}

	triOption := triConfig.NewTripleOption(opts...)

	_, ok = dp.ExporterMap().Load(url.ServiceKey())
	if !ok {
		panic("[DubboProtocol]" + url.Key() + "is not existing")
	}

	srv := triple.NewTripleServer(dp.serviceMap, triOption)
	dp.serverMap[url.Location] = srv
	srv.Start()
}

// GetProtocol get a single dubbo3 protocol.
func GetProtocol() protocol.Protocol {
	protocolOnce.Do(func() {
		dubboProtocol = NewDubboProtocol()
	})
	return dubboProtocol
}
