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
	"strconv"
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
	protocolbase "dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

type failoverClusterInvoker struct {
	base.BaseClusterInvoker
}

func newFailoverClusterInvoker(directory directory.Directory) protocolbase.Invoker {
	return &failoverClusterInvoker{
		BaseClusterInvoker: base.NewBaseClusterInvoker(directory),
	}
}

func (invoker *failoverClusterInvoker) Invoke(ctx context.Context, invocation protocolbase.Invocation) result.Result {
	var (
		res       result.Result
		invoked   []protocolbase.Invoker
		providers []string
		ivk       protocolbase.Invoker
	)

	invokers := invoker.Directory.List(invocation)
	if err := invoker.CheckInvokers(invokers, invocation); err != nil {
		return &result.RPCResult{Err: err}
	}

	methodName := invocation.ActualMethodName()
	retries := getRetries(invokers, methodName, invocation)
	loadBalance := base.GetLoadBalance(invokers[0], methodName)

	for i := 0; i <= retries; i++ {
		// Reselect before retry to avoid a change of candidate `invokers`.
		// NOTE: if `invokers` changed, then `invoked` also lose accuracy.
		if i > 0 {
			if err := invoker.CheckWhetherDestroyed(); err != nil {
				return &result.RPCResult{Err: err}
			}

			invokers = invoker.Directory.List(invocation)
			if err := invoker.CheckInvokers(invokers, invocation); err != nil {
				return &result.RPCResult{Err: err}
			}
		}
		ivk = invoker.DoSelect(loadBalance, invocation, invokers, invoked)
		if ivk == nil {
			continue
		}
		invoked = append(invoked, ivk)
		// DO INVOKE
		res = ivk.Invoke(ctx, invocation)
		if res.Error() != nil && !isBizError(res.Error()) {
			providers = append(providers, ivk.GetURL().Key())
			continue
		}
		return res
	}
	ip := common.GetLocalIp()
	invokerSvc := invoker.GetURL().Service()
	invokerUrl := invoker.Directory.GetURL()
	if ivk == nil {
		logger.Errorf("Failed to invoke the method %s of the service %s .No provider is available.", methodName, invokerSvc)
		return &result.RPCResult{
			Err: perrors.Errorf("Failed to invoke the method %s of the service %s .No provider is available because can't connect server.",
				methodName, invokerSvc),
		}
	}

	logger.Errorf(fmt.Sprintf("Failed to invoke the method %v in the service %v. "+
		"Tried %v times of the providers %v (%v/%v)from the registry %v on the consumer %v using the dubbo version %v. "+
		"Last error is %+v.", methodName, invokerSvc, retries, providers, len(providers), len(invokers),
		invokerUrl, ip, constant.Version, res.Error().Error()))

	return res
}

func isBizError(err error) bool {
	return triple_protocol.IsWireError(err) && triple_protocol.CodeOf(err) == triple_protocol.CodeBizError
}

func getRetries(invokers []protocolbase.Invoker, methodName string, invocation protocolbase.Invocation) int {
	// Todo(finalt) Temporarily solve the problem that the retries is not valid
	if retries, ok := invocation.GetAttachment(constant.RetriesKey); ok {
		if rInt, err := strconv.Atoi(retries); err == nil {
			return rInt
		}
	}
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
