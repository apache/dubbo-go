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
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
	dubbo3 "github.com/dubbogo/triple/pkg/triple"
	"google.golang.org/grpc"
)

const (
	// DUBBO3 is dubbo3 protocol name
	DUBBO3 = "dubbo3"
)

func init() {
	extension.SetProtocol(DUBBO3, GetProtocol)
}

var (
	dubbo3Protocol *Dubbo3Protocol
)

// It support dubbo protocol. It implements Protocol interface for dubbo protocol.
type Dubbo3Protocol struct {
	protocol.BaseProtocol
	serverLock sync.Mutex
	serverMap  map[string]*dubbo3.TripleServer // It is store relationship about serviceKey(group/interface:version) and ExchangeServer
}

// NewDubbo3Protocol create a dubbo protocol.
func NewDubbo3Protocol() *Dubbo3Protocol {
	return &Dubbo3Protocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*dubbo3.TripleServer),
	}
}

// Export export dubbo3 service.
func (dp *Dubbo3Protocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetUrl()
	serviceKey := url.ServiceKey()
	exporter := NewDubbo3Exporter(serviceKey, invoker, dp.ExporterMap())
	dp.SetExporterMap(serviceKey, exporter)
	logger.Infof("Export service: %s", url.String())
	// start server

	dp.openServer(url)
	return exporter
}

// Refer create dubbo3 service reference.
func (dp *Dubbo3Protocol) Refer(url *common.URL) protocol.Invoker {
	invoker, err := NewDubbo3Invoker(url)
	if err != nil {
		logger.Errorf("Refer url = %+v, with error = %s", *url, err.Error())
		return nil
	}
	dp.SetInvokers(invoker)
	logger.Infof("Refer service: %s", url.String())
	return invoker
}

// Destroy destroy dubbo3 service.
func (dp *Dubbo3Protocol) Destroy() {
	dp.BaseProtocol.Destroy()

	// stop server
	for key, server := range dp.serverMap {
		delete(dp.serverMap, key)
		server.Stop()
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

// openServer open a dubbo3 server
func (dp *Dubbo3Protocol) openServer(url *common.URL) {
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
	key := url.GetParam(constant.BEAN_NAME_KEY, "")
	service := config.GetProviderService(key)

	m, ok := reflect.TypeOf(service).MethodByName("SetProxyImpl")
	if !ok {
		panic("method SetProxyImpl is necessary for grpc service")
	}

	exporter, _ := dubbo3Protocol.ExporterMap().Load(url.ServiceKey())
	if exporter == nil {
		panic(fmt.Sprintf("no exporter found for servicekey: %v", url.ServiceKey()))
	}
	invoker := exporter.(protocol.Exporter).GetInvoker()
	if invoker == nil {
		panic(fmt.Sprintf("no invoker found for servicekey: %v", url.ServiceKey()))
	}

	in := []reflect.Value{reflect.ValueOf(service)}
	in = append(in, reflect.ValueOf(invoker))
	m.Func.Call(in)

	srv := dubbo3.NewTripleServer(url, service)

	dp.serverMap[url.Location] = srv

	srv.Start()
}

// GetProtocol get a single dubbo3 protocol.
func GetProtocol() protocol.Protocol {
	logger.Warn("GetProtocol")
	if dubbo3Protocol == nil {
		dubbo3Protocol = NewDubbo3Protocol()
	}
	return dubbo3Protocol
}
