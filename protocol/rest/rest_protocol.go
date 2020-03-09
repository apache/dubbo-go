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

package rest

import (
	"strings"
	"sync"
	"time"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
	_ "github.com/apache/dubbo-go/protocol/rest/rest_server"
)

var (
	restProtocol *RestProtocol
)

const REST = "rest"

func init() {
	extension.SetProtocol(REST, GetRestProtocol)
}

type RestProtocol struct {
	protocol.BaseProtocol
	serverMap  map[string]rest_interface.RestServer
	clientMap  map[rest_interface.RestOptions]rest_interface.RestClient
	serverLock sync.Mutex
	clientLock sync.Mutex
}

func NewRestProtocol() *RestProtocol {
	return &RestProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]rest_interface.RestServer),
		clientMap:    make(map[rest_interface.RestOptions]rest_interface.RestClient),
	}
}

func (rp *RestProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetUrl()
	serviceKey := url.ServiceKey()
	exporter := NewRestExporter(serviceKey, invoker, rp.ExporterMap())
	restServiceConfig := GetRestProviderServiceConfig(strings.TrimPrefix(url.Path, "/"))
	if restServiceConfig == nil {
		logger.Errorf("%s service doesn't has provider config", url.Path)
		return nil
	}
	rp.SetExporterMap(serviceKey, exporter)
	restServer := rp.getServer(url, restServiceConfig.Server)
	restServer.Deploy(invoker, restServiceConfig.RestMethodConfigsMap)
	return exporter
}

func (rp *RestProtocol) Refer(url common.URL) protocol.Invoker {
	// create rest_invoker
	var requestTimeout = config.GetConsumerConfig().RequestTimeout
	requestTimeoutStr := url.GetParam(constant.TIMEOUT_KEY, config.GetConsumerConfig().Request_Timeout)
	connectTimeout := config.GetConsumerConfig().ConnectTimeout
	if t, err := time.ParseDuration(requestTimeoutStr); err == nil {
		requestTimeout = t
	}
	restServiceConfig := GetRestConsumerServiceConfig(strings.TrimPrefix(url.Path, "/"))
	if restServiceConfig == nil {
		logger.Errorf("%s service doesn't has consumer config", url.Path)
		return nil
	}
	restOptions := rest_interface.RestOptions{RequestTimeout: requestTimeout, ConnectTimeout: connectTimeout}
	restClient := rp.getClient(restOptions, restServiceConfig.Client)
	invoker := NewRestInvoker(url, &restClient, restServiceConfig.RestMethodConfigsMap)
	rp.SetInvokers(invoker)
	return invoker
}

func (rp *RestProtocol) getServer(url common.URL, serverType string) rest_interface.RestServer {
	restServer, ok := rp.serverMap[url.Location]
	if !ok {
		_, ok := rp.ExporterMap().Load(url.ServiceKey())
		if !ok {
			panic("[RestProtocol]" + url.ServiceKey() + "is not existing")
		}
		rp.serverLock.Lock()
		restServer, ok = rp.serverMap[url.Location]
		if !ok {
			restServer = extension.GetNewRestServer(serverType)
			restServer.Start(url)
			rp.serverMap[url.Location] = restServer
		}
		rp.serverLock.Unlock()

	}
	return restServer
}

func (rp *RestProtocol) getClient(restOptions rest_interface.RestOptions, clientType string) rest_interface.RestClient {
	restClient, ok := rp.clientMap[restOptions]
	rp.clientLock.Lock()
	if !ok {
		restClient, ok = rp.clientMap[restOptions]
		if !ok {
			restClient = extension.GetNewRestClient(clientType, &restOptions)
			rp.clientMap[restOptions] = restClient
		}
	}
	rp.clientLock.Unlock()
	return restClient
}

func (rp *RestProtocol) Destroy() {
	// destroy rest_server
	rp.BaseProtocol.Destroy()
	for key, server := range rp.serverMap {
		server.Destroy()
		delete(rp.serverMap, key)
	}
	for key := range rp.clientMap {
		delete(rp.clientMap, key)
	}
}

func GetRestProtocol() protocol.Protocol {
	if restProtocol == nil {
		restProtocol = NewRestProtocol()
	}
	return restProtocol
}
