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
	"github.com/apache/dubbo-go/protocol/rest/client"
	_ "github.com/apache/dubbo-go/protocol/rest/client/client_impl"
	rest_config "github.com/apache/dubbo-go/protocol/rest/config"
	_ "github.com/apache/dubbo-go/protocol/rest/config/reader"
	"github.com/apache/dubbo-go/protocol/rest/server"
	_ "github.com/apache/dubbo-go/protocol/rest/server/server_impl"
)

var (
	restProtocol *RestProtocol
)

const REST = "rest"

// nolint
func init() {
	extension.SetProtocol(REST, GetRestProtocol)
}

// nolint
type RestProtocol struct {
	protocol.BaseProtocol
	serverLock sync.Mutex
	serverMap  map[string]server.RestServer
	clientLock sync.Mutex
	clientMap  map[client.RestOptions]client.RestClient
}

// NewRestProtocol returns a RestProtocol
func NewRestProtocol() *RestProtocol {
	return &RestProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]server.RestServer, 8),
		clientMap:    make(map[client.RestOptions]client.RestClient, 8),
	}
}

// Export export rest service
func (rp *RestProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetUrl()
	serviceKey := url.ServiceKey()
	exporter := NewRestExporter(serviceKey, invoker, rp.ExporterMap())
	restServiceConfig := rest_config.GetRestProviderServiceConfig(strings.TrimPrefix(url.Path, "/"))
	if restServiceConfig == nil {
		logger.Errorf("%s service doesn't has provider config", url.Path)
		return nil
	}
	rp.SetExporterMap(serviceKey, exporter)
	restServer := rp.getServer(url, restServiceConfig.Server)
	for _, methodConfig := range restServiceConfig.RestMethodConfigsMap {
		restServer.Deploy(methodConfig, server.GetRouteFunc(invoker, methodConfig))
	}
	return exporter
}

// Refer create rest service reference
func (rp *RestProtocol) Refer(url *common.URL) protocol.Invoker {
	// create rest_invoker
	var requestTimeout = config.GetConsumerConfig().RequestTimeout
	requestTimeoutStr := url.GetParam(constant.TIMEOUT_KEY, config.GetConsumerConfig().Request_Timeout)
	connectTimeout := config.GetConsumerConfig().ConnectTimeout
	if t, err := time.ParseDuration(requestTimeoutStr); err == nil {
		requestTimeout = t
	}
	restServiceConfig := rest_config.GetRestConsumerServiceConfig(strings.TrimPrefix(url.Path, "/"))
	if restServiceConfig == nil {
		logger.Errorf("%s service doesn't has consumer config", url.Path)
		return nil
	}
	restOptions := client.RestOptions{RequestTimeout: requestTimeout, ConnectTimeout: connectTimeout}
	restClient := rp.getClient(restOptions, restServiceConfig.Client)
	invoker := NewRestInvoker(url, &restClient, restServiceConfig.RestMethodConfigsMap)
	rp.SetInvokers(invoker)
	return invoker
}

// nolint
func (rp *RestProtocol) getServer(url *common.URL, serverType string) server.RestServer {
	restServer, ok := rp.serverMap[url.Location]
	if ok {
		return restServer
	}
	_, ok = rp.ExporterMap().Load(url.ServiceKey())
	if !ok {
		panic("[RestProtocol]" + url.ServiceKey() + "is not existing")
	}
	rp.serverLock.Lock()
	defer rp.serverLock.Unlock()
	restServer, ok = rp.serverMap[url.Location]
	if ok {
		return restServer
	}
	restServer = extension.GetNewRestServer(serverType)
	restServer.Start(url)
	rp.serverMap[url.Location] = restServer
	return restServer
}

// nolint
func (rp *RestProtocol) getClient(restOptions client.RestOptions, clientType string) client.RestClient {
	restClient, ok := rp.clientMap[restOptions]
	if ok {
		return restClient
	}
	rp.clientLock.Lock()
	defer rp.clientLock.Unlock()
	restClient, ok = rp.clientMap[restOptions]
	if ok {
		return restClient
	}
	restClient = extension.GetNewRestClient(clientType, &restOptions)
	rp.clientMap[restOptions] = restClient
	return restClient
}

// Destroy destroy rest service
func (rp *RestProtocol) Destroy() {
	// destroy rest_server
	rp.BaseProtocol.Destroy()
	for key, tmpServer := range rp.serverMap {
		tmpServer.Destroy()
		delete(rp.serverMap, key)
	}
	for key := range rp.clientMap {
		delete(rp.clientMap, key)
	}
}

// GetRestProtocol get a rest protocol
func GetRestProtocol() protocol.Protocol {
	if restProtocol == nil {
		restProtocol = NewRestProtocol()
	}
	return restProtocol
}
