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
	"context"
	"fmt"
	"sync"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/remoting"
	"github.com/apache/dubbo-go/remoting/getty"
	"github.com/opentracing/opentracing-go"
)

// dubbo protocol constant
const (
	// DUBBO ...
	DUBBO = "dubbo"
)

var (
	exchangeClientMap *sync.Map = new(sync.Map)
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
	serverMap  map[string]*remoting.ExchangeServer
	serverLock sync.Mutex
}

// NewDubboProtocol ...
func NewDubboProtocol() *DubboProtocol {
	return &DubboProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*remoting.ExchangeServer),
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
	//var requestTimeout = config.GetConsumerConfig().RequestTimeout
	//
	//requestTimeoutStr := url.GetParam(constant.TIMEOUT_KEY, config.GetConsumerConfig().Request_Timeout)
	//if t, err := time.ParseDuration(requestTimeoutStr); err == nil {
	//	requestTimeout = t
	//}

	//invoker := NewDubboInvoker(url, NewClient(Options{
	//	ConnectTimeout: config.GetConsumerConfig().ConnectTimeout,
	//	RequestTimeout: requestTimeout,
	//}))
	exchangeClient := getExchangeClient(url)
	if exchangeClient == nil {
		return nil
	}
	invoker := NewDubboInvoker(url, exchangeClient)
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
			handler := func(invocation *invocation.RPCInvocation) protocol.RPCResult {
				return doHandleRequest(invocation)
			}
			srv := remoting.NewExchangeServer(url, getty.NewServer(url, handler))
			dp.serverMap[url.Location] = srv
			srv.Start()
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

func doHandleRequest(rpcInvocation *invocation.RPCInvocation) protocol.RPCResult {
	exporter, _ := dubboProtocol.ExporterMap().Load(rpcInvocation.ServiceKey())
	result := protocol.RPCResult{}
	if exporter == nil {
		err := fmt.Errorf("don't have this exporter, key: %s", rpcInvocation.ServiceKey())
		logger.Errorf(err.Error())
		result.Err = err
		//reply(session, p, hessian.PackageResponse)
		return result
	}
	invoker := exporter.(protocol.Exporter).GetInvoker()
	if invoker != nil {
		// FIXME
		ctx := rebuildCtx(rpcInvocation)

		invokeResult := invoker.Invoke(ctx, rpcInvocation)
		if err := invokeResult.Error(); err != nil {
			result.Err = invokeResult.Error()
			//p.Header.ResponseStatus = hessian.Response_OK
			//p.Body = hessian.NewResponse(nil, err, result.Attachments())
		} else {
			result.Rest = invokeResult.Result()
			//p.Header.ResponseStatus = hessian.Response_OK
			//p.Body = hessian.NewResponse(res, nil, result.Attachments())
		}
	} else {
		result.Err = fmt.Errorf("don't have the invoker, key: %s", rpcInvocation.ServiceKey())
	}
	return result
}

func getExchangeClient(url common.URL) *remoting.ExchangeClient {
	clientTmp, ok := exchangeClientMap.Load(url.Location)
	if !ok {
		exchangeClientTmp := remoting.NewExchangeClient(url, getty.NewClient(getty.Options{
			ConnectTimeout: config.GetConsumerConfig().ConnectTimeout,
		}), config.GetConsumerConfig().ConnectTimeout)
		if exchangeClientTmp != nil {
			exchangeClientMap.Store(url.Location, exchangeClientTmp)
		}

		return exchangeClientTmp
	}
	exchangeClient, ok := clientTmp.(*remoting.ExchangeClient)
	if !ok {
		exchangeClientTmp := remoting.NewExchangeClient(url, getty.NewClient(getty.Options{
			ConnectTimeout: config.GetConsumerConfig().ConnectTimeout,
		}), config.GetConsumerConfig().ConnectTimeout)
		if exchangeClientTmp != nil {
			exchangeClientMap.Store(url.Location, exchangeClientTmp)
		}
		return exchangeClientTmp
	}
	return exchangeClient
}

// rebuildCtx rebuild the context by attachment.
// Once we decided to transfer more context's key-value, we should change this.
// now we only support rebuild the tracing context
func rebuildCtx(inv *invocation.RPCInvocation) context.Context {
	ctx := context.WithValue(context.Background(), "attachment", inv.Attachments())

	// actually, if user do not use any opentracing framework, the err will not be nil.
	spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.TextMap,
		opentracing.TextMapCarrier(inv.Attachments()))
	if err == nil {
		ctx = context.WithValue(ctx, constant.TRACING_REMOTE_SPAN_CTX, spanCtx)
	}
	return ctx
}
