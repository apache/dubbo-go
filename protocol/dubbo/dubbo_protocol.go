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

package dubbo

import (
	"fmt"
	hessian "github.com/apache/dubbo-go-hessian2"
	"github.com/apache/dubbo-go/remoting"
	"github.com/apache/dubbo-go/remoting/getty"
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
)

// dubbo protocol constant
const (
	// DUBBO ...
	//DUBBO = "dubbo"
)

func init() {
	extension.SetProtocol(DUBBO, GetProtocol)
}

var (
	dubboProtocol *DubboProtocol
)

// DubboProtocol ...
type DubboProtocol struct {
	protocol.BaseProtocol
	serverMap  map[string]*getty.Server
	serverLock sync.Mutex
}

// NewDubboProtocol ...
func NewDubboProtocol() *DubboProtocol {
	return &DubboProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*getty.Server),
	}
}

// Export ...
func (dp *DubboProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetUrl()
	serviceKey := url.ServiceKey()
	exporter := NewDubboExporter(serviceKey, invoker, dp.ExporterMap())
	dp.SetExporterMap(serviceKey, exporter)
	logger.Infof("Export service: %s", url.String())

	// start server
	dp.openServer(url)
	return exporter
}

// Refer ...
func (dp *DubboProtocol) Refer(url common.URL) protocol.Invoker {
	//default requestTimeout
	var requestTimeout = config.GetConsumerConfig().RequestTimeout

	requestTimeoutStr := url.GetParam(constant.TIMEOUT_KEY, config.GetConsumerConfig().Request_Timeout)
	if t, err := time.ParseDuration(requestTimeoutStr); err == nil {
		requestTimeout = t
	}

	invoker := NewDubboInvoker(url, getty.NewClient(getty.Options{
		ConnectTimeout: config.GetConsumerConfig().ConnectTimeout,
		RequestTimeout: requestTimeout,
	}))
	dp.SetInvokers(invoker)
	logger.Infof("Refer service: %s", url.String())
	return invoker
}

// Destroy ...
func (dp *DubboProtocol) Destroy() {
	logger.Infof("DubboProtocol destroy.")

	dp.BaseProtocol.Destroy()

	// stop server
	for key, server := range dp.serverMap {
		delete(dp.serverMap, key)
		server.Stop()
	}
}

func (dp *DubboProtocol) Reply(url *common.URL, invocation protocol.Invocation) protocol.Result {
	return nil
}

func (dp *DubboProtocol) openServer(url common.URL) {
	_, ok := dp.serverMap[url.Location]
	if !ok {
		_, ok := dp.ExporterMap().Load(url.ServiceKey())
		if !ok {
			panic("[DubboProtocol]" + url.Key() + "is not existing")
		}

		dp.serverLock.Lock()
		_, ok = dp.serverMap[url.Location]
		if !ok {
			srv := getty.NewServer(reply)
			dp.serverMap[url.Location] = srv
			srv.Start(url)
		}
		dp.serverLock.Unlock()
	}
}

// GetProtocol ...
func GetProtocol() protocol.Protocol {
	if dubboProtocol == nil {
		dubboProtocol = NewDubboProtocol()
	}
	return dubboProtocol
}

func reply(url *common.URL, invocation protocol.Invocation) protocol.Result {
	exporter, _ := dubboProtocol.ExporterMap().Load(u.ServiceKey())
	if exporter == nil {
		err := fmt.Errorf("don't have this exporter, key: %s", u.ServiceKey())
		logger.Errorf(err.Error())
		p.Header.ResponseStatus = hessian.Response_OK
		p.Body = err
		reply(session, p, hessian.PackageResponse)
		return &protocol.RPCResult{
			Err:err,
		}
	}
	invoker := exporter.(protocol.Exporter).GetInvoker()
	if invoker != nil {
		attachments := p.Body.(map[string]interface{})["attachments"].(map[string]string)
		attachments[constant.LOCAL_ADDR] = session.LocalAddr()
		attachments[constant.REMOTE_ADDR] = session.RemoteAddr()

		args := p.Body.(map[string]interface{})["args"].([]interface{})
		inv := invocation.NewRPCInvocation(p.Service.Method, args, attachments)

		ctx := rebuildCtx(inv)

		result := invoker.Invoke(ctx, inv)
		if err := result.Error(); err != nil {
			p.Header.ResponseStatus = hessian.Response_OK
			p.Body = hessian.NewResponse(nil, err, result.Attachments())
		} else {
			res := result.Result()
			p.Header.ResponseStatus = hessian.Response_OK
			p.Body = hessian.NewResponse(res, nil, result.Attachments())
		}
		return result;
	}
	return &protocol.RPCResult{
		Err:fmt.Errorf("don't have the invoker, key: %s", url.ServiceKey()),
	}
}
