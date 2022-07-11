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

package mq

import (
	"sync"
)

import (
	"github.com/dubbogo/triple/pkg/triple"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/getty"
)

const RocketMQ = "rocketmq"

var protocolOnce sync.Once

func init() {
	extension.SetProtocol(RocketMQ, GetProtocol)
	protocolOnce = sync.Once{}
}

var (
	rocketmqProtocol *RocketMQProtocol
)

// RocketMQProtocol supports dubbo 3.0 protocol. It implements Protocol interface for dubbo protocol.
type RocketMQProtocol struct {
	protocol.BaseProtocol
	serverLock sync.Mutex

	serviceMap *sync.Map                       // serviceMap is used to export multiple service by one server
	serverMap  map[string]*triple.TripleServer // serverMap stores all exported server
}

func NewRocketMQProtocol() *RocketMQProtocol {
	return &RocketMQProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
	}
}

func (rp *RocketMQProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetURL()
	serviceKey := url.ServiceKey()
	// todo
	exporter := NewRocketMQExporter(serviceKey, invoker, nil, nil)
	rp.SetExporterMap(serviceKey, exporter)
	logger.Infof("[rocketmq Protocol] Export service: %s", url.String())

	// start server
	rp.openServer(url)
	return exporter
}

func (rp *RocketMQProtocol) Refer(url *common.URL) protocol.Invoker {
	invoker, err := NewRocketMQInvoker(url)
	if err != nil {
		logger.Errorf("rocketmq protocol Refer url = %+v, with error = %s", url, err.Error())
		return nil
	}
	rp.SetInvokers(invoker)
	logger.Infof("[Rocketmq Protocol] Refer service: %s", url.String())
	return invoker
}

// Destroy rocketmq service.
func (rp *RocketMQProtocol) Destroy() {
	logger.Infof("RocketMQ Protocol destroy.")

	rp.BaseProtocol.Destroy()

	// stop server
	for key, server := range rp.serverMap {
		delete(rp.serverMap, key)
		server.Stop()
	}
}

func (rp *RocketMQProtocol) openServer(url *common.URL) {
	_, ok := rp.serverMap[url.Location]
	if !ok {
		_, ok := rp.ExporterMap().Load(url.ServiceKey())
		if !ok {
			panic("[RocketMQProtocol]" + url.Key() + "is not existing")
		}

		rp.serverLock.Lock()
		_, ok = rp.serverMap[url.Location]
		if !ok {
			handler := func(invocation *invocation.RPCInvocation) protocol.RPCResult {
				return doHandleRequest(invocation)
			}
			srv := remoting.NewExchangeServer(url, getty.NewServer(url, handler))
			rp.serverMap[url.Location] = srv
			srv.Start()
		}
		rp.serverLock.Unlock()
	}
}

// GetProtocol get a single rocketmq protocol.
func GetProtocol() protocol.Protocol {
	if rocketmqProtocol == nil {
		rocketmqProtocol = NewRocketMQProtocol()
	}
	return rocketmqProtocol
}
