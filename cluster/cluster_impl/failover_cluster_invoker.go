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
	"context"
	"fmt"
	"strconv"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/cluster"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
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

// nolint
func (invoker *failoverClusterInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	var (
		result    protocol.Result
		invoked   []protocol.Invoker
		providers []string
		ivk       protocol.Invoker
	)

	invokers := invoker.directory.List(invocation)
	if err := invoker.checkInvokers(invokers, invocation); err != nil {
		return &protocol.RPCResult{Err: err}
	}

	methodName := invocation.MethodName()
	retries := getRetries(invokers, methodName)
	loadBalance := getLoadBalance(invokers[0], invocation)

	for i := 0; i <= retries; i++ {
		//Reselect before retry to avoid a change of candidate `invokers`.
		//NOTE: if `invokers` changed, then `invoked` also lose accuracy.
		if i > 0 {
			if err := invoker.checkWhetherDestroyed(); err != nil {
				return &protocol.RPCResult{Err: err}
			}

			invokers = invoker.directory.List(invocation)
			if err := invoker.checkInvokers(invokers, invocation); err != nil {
				return &protocol.RPCResult{Err: err}
			}
		}
		ivk = invoker.doSelect(loadBalance, invocation, invokers, invoked)
		if ivk == nil {
			continue
		}
		invoked = append(invoked, ivk)
		//DO INVOKE
		result = ivk.Invoke(ctx, invocation)
		if result.Error() != nil {
			providers = append(providers, ivk.GetUrl().Key())
			continue
		}
		return result
	}
	ip := common.GetLocalIp()
	invokerSvc := invoker.GetUrl().Service()
	invokerUrl := invoker.directory.GetUrl()
	if ivk == nil {
		logger.Errorf("Failed to invoke the method %s of the service %s .No provider is available.", methodName, invokerSvc)
		return &protocol.RPCResult{
			Err: perrors.Errorf("Failed to invoke the method %s of the service %s .No provider is available because can't connect server.",
				methodName, invokerSvc),
		}
	}

	return &protocol.RPCResult{
		Err: perrors.Wrap(result.Error(), fmt.Sprintf("Failed to invoke the method %v in the service %v. "+
			"Tried %v times of the providers %v (%v/%v)from the registry %v on the consumer %v using the dubbo version %v. "+
			"Last error is %+v.", methodName, invokerSvc, retries, providers, len(providers), len(invokers),
			invokerUrl, ip, constant.Version, result.Error().Error()),
		)}
}

func getRetries(invokers []protocol.Invoker, methodName string) int {
	if len(invokers) <= 0 {
		return constant.DEFAULT_RETRIES_INT
	}

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

	if retries > len(invokers) {
		retries = len(invokers)
	}
	return retries
}
