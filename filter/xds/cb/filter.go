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

package cb

import (
	"context"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds"
	"dubbo.apache.org/dubbo-go/v3/xds/client"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

// this should be executed before users set their own Tracer
func init() {
	extension.SetFilter(constant.XdsCircuitBreakerKey, newCircuitBreakerFilter)
}

// if you wish to using opentracing, please add the this filter into your filter attribute in your configure file.
// notice that this could be used in both client-side and server-side.
type circuitBreakerFilter struct {
	client xds.XDSWrapperClient
}

func (cb *circuitBreakerFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	url := invoker.GetURL()
	rejectedExeHandler := url.GetParam(constant.DefaultKey, constant.DefaultKey)
	clusterUpdate, err := cb.getClusterUpdate(url)
	if err != nil {
		logger.Errorf("xds circuitBreakerFilter get request counter fail", err)
		return nil
	}
	counter := client.GetClusterRequestsCounter(clusterUpdate.ClusterName, clusterUpdate.EDSServiceName)
	if err := counter.StartRequest(*clusterUpdate.MaxRequests); err != nil {
		rejectedExecutionHandler, err := extension.GetRejectedExecutionHandler(rejectedExeHandler)
		if err != nil {
			logger.Warn(err)
		} else {
			return rejectedExecutionHandler.RejectedExecution(url, invocation)
		}
	}
	return invoker.Invoke(ctx, invocation)
}

func (cb *circuitBreakerFilter) OnResponse(ctx context.Context, result protocol.Result,
	invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	url := invoker.GetURL()
	clusterUpdate, err := cb.getClusterUpdate(url)
	if err != nil {
		logger.Errorf("xds circuitBreakerFilter get request counter fail", err)
		return nil
	}
	counter := client.GetClusterRequestsCounter(clusterUpdate.ClusterName, clusterUpdate.EDSServiceName)
	counter.EndRequest()
	return result
}

var circuitBreakerFilterInstance filter.Filter

func newCircuitBreakerFilter() filter.Filter {
	if circuitBreakerFilterInstance == nil {
		circuitBreakerFilterInstance = &circuitBreakerFilter{
			client: xds.GetXDSWrappedClient(),
		}
	}
	return circuitBreakerFilterInstance
}

func (cb *circuitBreakerFilter) getClusterUpdate(url *common.URL) (resource.ClusterUpdate, error) {
	hostAddr, err := cb.client.GetHostAddrByServiceUniqueKey(common.GetSubscribeName(url))
	if err != nil {
		logger.Errorf("xds circuitBreakerFilter get GetHostAddrByServiceUniqueKey fail", err)
		return resource.ClusterUpdate{}, err
	}
	clusterUpdate := cb.client.GetClusterUpdateIgnoreVersion(hostAddr)
	return clusterUpdate, nil
}
