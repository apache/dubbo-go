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
	"strconv"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/common/utils"
	"github.com/apache/dubbo-go/protocol"
)

type failoverClusterInvoker struct {
	baseClusterInvoker
}

func newFailoverClusterInvoker(directory cluster.Directory) protocol.Invoker {
	return &failoverClusterInvoker{
		baseClusterInvoker: newBaseClusterInvoker(directory),
	}
}

func (invoker *failoverClusterInvoker) Invoke(invocation protocol.Invocation) protocol.Result {

	invokers := invoker.directory.List(invocation)
	err := invoker.checkInvokers(invokers, invocation)

	if err != nil {
		return &protocol.RPCResult{Err: err}
	}

	loadbalance := getLoadBalance(invokers[0], invocation)

	methodName := invocation.MethodName()
	url := invokers[0].GetUrl()

	//get reties
	retriesConfig := url.GetParam(constant.RETRIES_KEY, constant.DEFAULT_RETRIES)

	//Get the service method loadbalance config if have
	if v := url.GetMethodParam(methodName, constant.RETRIES_KEY, ""); len(v) != 0 {
		retriesConfig = v
	}
	retries, err := strconv.Atoi(retriesConfig)
	if err != nil || retries < 0 {
		logger.Error("Your retries config is invalid,pls do a check. And will use the default retries configuration instead.")
		retries = constant.DEFAULT_RETRIES_INT
	}
	invoked := []protocol.Invoker{}
	providers := []string{}
	var result protocol.Result
	for i := 0; i <= retries; i++ {
		//Reselect before retry to avoid a change of candidate `invokers`.
		//NOTE: if `invokers` changed, then `invoked` also lose accuracy.
		if i > 0 {
			err := invoker.checkWhetherDestroyed()
			if err != nil {
				return &protocol.RPCResult{Err: err}
			}
			invokers = invoker.directory.List(invocation)
			err = invoker.checkInvokers(invokers, invocation)
			if err != nil {
				return &protocol.RPCResult{Err: err}
			}
		}
		ivk := invoker.doSelect(loadbalance, invocation, invokers, invoked)
		invoked = append(invoked, ivk)
		//DO INVOKE
		result = ivk.Invoke(invocation)
		if result.Error() != nil {
			providers = append(providers, ivk.GetUrl().Key())
			continue
		} else {
			return result
		}
	}
	ip, _ := utils.GetLocalIP()
	return &protocol.RPCResult{Err: perrors.Errorf("Failed to invoke the method %v in the service %v. Tried %v times of "+
		"the providers %v (%v/%v)from the registry %v on the consumer %v using the dubbo version %v. Last error is %v.",
		methodName, invoker.GetUrl().Service(), retries, providers, len(providers), len(invokers), invoker.directory.GetUrl(), ip, constant.Version, result.Error().Error(),
	)}
}
