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

package failover

import (
	"context"
	"fmt"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type failoverClusterInvoker struct {
	base.BaseClusterInvoker
}

func newFailoverClusterInvoker(directory directory.Directory) protocol.Invoker {
	return &failoverClusterInvoker{
		BaseClusterInvoker: base.NewBaseClusterInvoker(directory),
	}
}

func (invoker *failoverClusterInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	var (
		result    protocol.Result
		invoked   []protocol.Invoker
		providers []string
		ivk       protocol.Invoker
	)

	invokers := invoker.Directory.List(invocation)
	if err := invoker.CheckInvokers(invokers, invocation); err != nil {
		return &protocol.RPCResult{Err: err}
	}

	methodName := invocation.ActualMethodName()
	retries := getRetries(invokers, methodName)
	loadBalance := base.GetLoadBalance(invokers[0], methodName)

	for i := 0; i <= retries; i++ {
		// Reselect before retry to avoid a change of candidate `invokers`.
		// NOTE: if `invokers` changed, then `invoked` also lose accuracy.
		if i > 0 {
			if err := invoker.CheckWhetherDestroyed(); err != nil {
				return &protocol.RPCResult{Err: err}
			}

			invokers = invoker.Directory.List(invocation)
			if err := invoker.CheckInvokers(invokers, invocation); err != nil {
				return &protocol.RPCResult{Err: err}
			}
		}
		ivk = invoker.DoSelect(loadBalance, invocation, invokers, invoked)
		if ivk == nil {
			continue
		}
		invoked = append(invoked, ivk)
		// DO INVOKE
		result = ivk.Invoke(ctx, invocation)
		if result.Error() != nil {
			providers = append(providers, ivk.GetURL().Key())
			continue
		}
		return result
	}
	ip := common.GetLocalIp()
	invokerSvc := invoker.GetURL().Service()
	invokerUrl := invoker.Directory.GetURL()
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
		),
	}
}

func getRetries(invokers []protocol.Invoker, methodName string) int {
	if len(invokers) <= 0 {
		return constant.DefaultRetriesInt
	}
	url := invokers[0].GetURL()

	retries := url.GetMethodParamIntValue(methodName, constant.RetriesKey,
		url.GetParamByIntValue(constant.RetriesKey, constant.DefaultRetriesInt))

	if retries < 0 {
		return constant.DefaultRetriesInt
	}
	return retries
}
