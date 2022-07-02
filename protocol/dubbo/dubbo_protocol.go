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
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/opentracing/opentracing-go"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	"dubbo.apache.org/dubbo-go/v3/remoting/getty"
)

const (
	// DUBBO is dubbo protocol name
	DUBBO = "dubbo"
)

var (
	// Make the connection can be shared.
	// It will create one connection for one address (ip+port)
	exchangeClientMap = new(sync.Map)
	exchangeLock      = new(sync.Map)
)

func init() {
	extension.SetProtocol(DUBBO, GetProtocol)
}

var dubboProtocol *DubboProtocol

// DubboProtocol supports dubbo protocol. It implements Protocol interface for dubbo protocol.
type DubboProtocol struct {
	protocol.BaseProtocol
	// It is store relationship about serviceKey(group/interface:version) and ExchangeServer
	// The ExchangeServer is introduced to replace of Server. Because Server is depend on getty directly.
	serverMap  map[string]*remoting.ExchangeServer
	serverLock sync.Mutex
}

// NewDubboProtocol create a dubbo protocol.
func NewDubboProtocol() *DubboProtocol {
	return &DubboProtocol{
		BaseProtocol: protocol.NewBaseProtocol(),
		serverMap:    make(map[string]*remoting.ExchangeServer),
	}
}

// Export export dubbo service.
func (dp *DubboProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	url := invoker.GetURL()
	serviceKey := url.ServiceKey()
	exporter := NewDubboExporter(serviceKey, invoker, dp.ExporterMap())
	dp.SetExporterMap(serviceKey, exporter)
	logger.Infof("[DUBBO Protocol] Export service: %s", url.String())
	// start server
	dp.openServer(url)
	return exporter
}

// Refer create dubbo service reference.
func (dp *DubboProtocol) Refer(url *common.URL) protocol.Invoker {
	exchangeClient := getExchangeClient(url)
	if exchangeClient == nil {
		logger.Warnf("can't dial the server: %+v", url.Location)
		return nil
	}
	invoker := NewDubboInvoker(url, exchangeClient)
	dp.SetInvokers(invoker)
	logger.Infof("[DUBBO Protocol] Refer service: %s", url.String())
	return invoker
}

// Destroy destroy dubbo service.
func (dp *DubboProtocol) Destroy() {
	logger.Infof("DubboProtocol destroy.")

	dp.BaseProtocol.Destroy()

	// stop server
	for key, server := range dp.serverMap {
		delete(dp.serverMap, key)
		server.Stop()
	}
}

func (dp *DubboProtocol) openServer(url *common.URL) {
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

// GetProtocol get a single dubbo protocol.
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
		// reply(session, p, hessian.PackageResponse)
		return result
	}
	invoker := exporter.(protocol.Exporter).GetInvoker()
	if invoker != nil {
		// FIXME
		ctx := rebuildCtx(rpcInvocation)

		invokeResult := invoker.Invoke(ctx, rpcInvocation)
		if err := invokeResult.Error(); err != nil {
			result.Err = invokeResult.Error()
			// p.Header.ResponseStatus = hessian.Response_OK
			// p.Body = hessian.NewResponse(nil, err, result.Attachments())
		} else {
			result.Rest = invokeResult.Result()
			// p.Header.ResponseStatus = hessian.Response_OK
			// p.Body = hessian.NewResponse(res, nil, result.Attachments())
		}
		result.Attrs = invokeResult.Attachments()
	} else {
		result.Err = fmt.Errorf("don't have the invoker, key: %s", rpcInvocation.ServiceKey())
	}
	return result
}

func getExchangeClient(url *common.URL) *remoting.ExchangeClient {
	clientTmp, ok := exchangeClientMap.Load(url.Location)
	if !ok {
		var exchangeClientTmp *remoting.ExchangeClient
		func() {
			// lock for NewExchangeClient and store into map.
			_, loaded := exchangeLock.LoadOrStore(url.Location, 0x00)
			// unlock
			defer exchangeLock.Delete(url.Location)
			if loaded {
				// retry for 5 times.
				for i := 0; i < 5; i++ {
					if clientTmp, ok = exchangeClientMap.Load(url.Location); ok {
						break
					} else {
						// if cannot get, sleep a while.
						time.Sleep(time.Duration(i*100) * time.Millisecond)
					}
				}
				return
			}

			// todo set by config
			exchangeClientTmp = remoting.NewExchangeClient(url, getty.NewClient(getty.Options{
				ConnectTimeout: 3 * time.Second,
				RequestTimeout: 3 * time.Second,
			}), 3*time.Second, false)
			// input store
			if exchangeClientTmp != nil {
				exchangeClientMap.Store(url.Location, exchangeClientTmp)
			}
		}()
		if exchangeClientTmp != nil {
			return exchangeClientTmp
		}
	}
	// cannot dial the server
	if clientTmp == nil {
		return nil
	}
	exchangeClient := clientTmp.(*remoting.ExchangeClient)
	exchangeClient.IncreaseActiveNumber()
	return exchangeClient
}

// rebuildCtx rebuild the context by attachment.
// Once we decided to transfer more context's key-value, we should change this.
// now we only support rebuild the tracing context
func rebuildCtx(inv *invocation.RPCInvocation) context.Context {
	ctx := context.WithValue(context.Background(), constant.DubboCtxKey("attachment"), inv.Attachments())

	// actually, if user do not use any opentracing framework, the err will not be nil.
	spanCtx, err := opentracing.GlobalTracer().Extract(opentracing.TextMap,
		opentracing.TextMapCarrier(filterContext(inv.Attachments())))
	if err == nil {
		ctx = context.WithValue(ctx, constant.DubboCtxKey(constant.TracingRemoteSpanCtx), spanCtx)
	}
	return ctx
}
