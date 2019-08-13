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

package cluster_impl

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
)

/**
 * When invoke fails, log the error message and ignore this error by returning an empty Result.
 * Usually used to write audit logs and other operations
 *
 * <a href="http://en.wikipedia.org/wiki/Fail-safe">Fail-safe</a>
 *
 */
type failsafeClusterInvoker struct {
	baseClusterInvoker
}

func newFailsafeClusterInvoker(directory cluster.Directory) protocol.Invoker {
	return &failsafeClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(directory),
	}
}

func (invoker *failsafeClusterInvoker) Invoke(invocation protocol.Invocation) protocol.Result {
	invokers := invoker.directory.List(invocation)

	err := invoker.checkInvokers(invokers, invocation)
	if err != nil {
		return &protocol.RPCResult{}
	}

	url := invokers[0].GetUrl()
	methodName := invocation.MethodName()
	//Get the service loadbalance config
	lb := url.GetParam(constant.LOADBALANCE_KEY, constant.DEFAULT_LOADBALANCE)
	//Get the service method loadbalance config if have
	if v := url.GetMethodParam(methodName, constant.LOADBALANCE_KEY, ""); v != "" {
		lb = v
	}
	loadbalance := extension.GetLoadbalance(lb)

	invoked := make([]protocol.Invoker, 0)
	var result protocol.Result

	ivk := invoker.doSelect(loadbalance, invocation, invokers, invoked)
	//DO INVOKE
	result = ivk.Invoke(invocation)
	if result.Error() != nil {
		// ignore
		logger.Errorf("Failsafe ignore exception: %v.\n", result.Error().Error())
		return &protocol.RPCResult{}
	}
	return result
}
