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
	"fmt"
	"reflect"
	"sync"
)

import (
	tripleCommon "github.com/dubbogo/triple/pkg/common"
	triConfig "github.com/dubbogo/triple/pkg/config"
	"github.com/dubbogo/triple/pkg/triple"
	"google.golang.org/grpc"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
)

var protocolOnce sync.Once

func init() {
	extension.SetProtocol(tripleCommon.TRIPLE, GetProtocol)
	protocolOnce = sync.Once{}
}

var (
	dubboProtocol *DubboProtocol
)

// It support dubbo protocol. It implements Protocol interface for dubbo protocol.
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
	logger.Infof("Export service: %s", url.String())

	key := url.GetParam(constant.BEAN_NAME_KEY, "")
	service := config.GetProviderService(key)

	serializationType := url.GetParam(constant.SERIALIZATION_KEY, constant.PROTOBUF_SERIALIZATION)
	var triSerializationType tripleCommon.TripleSerializerName
	switch serializationType {
	case constant.PROTOBUF_SERIALIZATION:
		m, ok := reflect.TypeOf(service).MethodByName("SetProxyImpl")
		if !ok {
			panic("method SetProxyImpl is necessary for triple service")
		}
		if invoker == nil {
			panic(fmt.Sprintf("no invoker found for servicekey: %v", url.ServiceKey()))
		}
		in := []reflect.Value{reflect.ValueOf(service)}
		in = append(in, reflect.ValueOf(invoker))
		m.Func.Call(in)
		triSerializationType = tripleCommon.PBSerializerName
	case constant.HESSIAN2_SERIALIZATION:
		tripleService := Dubbo3HessianService{}
		tripleService.SetProxyImpl(invoker)
		service = &tripleService
		triSerializationType = tripleCommon.TripleHessianWrapperSerializerName
	default:
		panic(fmt.Sprintf("unsupport serialization = %s", serializationType))
	}

	dp.serviceMap.Store(url.GetParam(constant.INTERFACE_KEY, ""), service)

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
	logger.Infof("Refer service: %s", url.String())
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
	SetProxyImpl(impl protocol.Invoker)
	// GetProxyImpl gets proxy.
	GetProxyImpl() protocol.Invoker
	// ServiceDesc gets an RPC service's specification.
	ServiceDesc() *grpc.ServiceDesc
}

type Dubbo3HessianService struct {
	proxyImpl protocol.Invoker
}

func (d *Dubbo3HessianService) SetProxyImpl(impl protocol.Invoker) {
	d.proxyImpl = impl
}

func (d *Dubbo3HessianService) GetProxyImpl() protocol.Invoker {
	return d.proxyImpl
}

func (d *Dubbo3HessianService) ServiceDesc() *grpc.ServiceDesc {
	return nil
}

func (d *Dubbo3HessianService) Reference() string {
	return ""
}

// openServer open a dubbo3 server, if there is already a service using the same protocol, it returns directly.
func (dp *DubboProtocol) openServer(url *common.URL, tripleSerializationType tripleCommon.TripleSerializerName) {
	_, ok := dp.serverMap[url.Location]
	if ok {
		return
	}
	_, ok = dp.ExporterMap().Load(url.ServiceKey())
	if !ok {
		panic("[DubboProtocol]" + url.Key() + "is not existing")
	}

	dp.serverLock.Lock()
	defer dp.serverLock.Unlock()
	_, ok = dp.serverMap[url.Location]
	if ok {
		return
	}

	triOption := triConfig.NewTripleOption(
		triConfig.WithSerializerType(tripleSerializationType),
	)
	srv := triple.NewTripleServer(url, dp.serviceMap, triOption)
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
