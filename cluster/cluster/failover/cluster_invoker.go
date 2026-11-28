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
)

import (
	cluster "dubbo.apache.org/dubbo-go/v3/cluster/cluster"
	"dubbo.apache.org/dubbo-go/v3/cluster/cluster/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/directory"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	dubboLogger "dubbo.apache.org/dubbo-go/v3/logger"
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
		retryErr  error
		invoked   []protocolbase.Invoker
		providers []string
		ivk       protocolbase.Invoker
	)

	invokers := invoker.Directory.List(invocation)
	if err := invoker.CheckInvokersWithContext(ctx, invokers, invocation); err != nil {
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
			if err := invoker.CheckInvokersWithContext(ctx, invokers, invocation); err != nil {
				return &result.RPCResult{Err: err}
			}
			if retryErr != nil {
				if observer := cluster.DiagnosticsFromContext(ctx, invocation); observer != nil {
					observer.Retry(i+1, retryErr)
				}
				retryErr = nil
			}
		}
		ivk = invoker.DoSelectWithContext(ctx, loadBalance, invocation, invokers, invoked, i+1)
		if ivk == nil {
			continue
		}
		invoked = append(invoked, ivk)
		// The metrics filter counts an attempt of the invocation marked in its
		// context as a retry. Attempts of the same invocation inside an enclosing
		// retry, such as a group failover inside a multi-group failover, inherit
		// the mark; other RPCs started with this context do not match it.
		attemptCtx := ctx
		if len(invoked) > 1 {
			attemptCtx = protocolbase.WithRetryAttempt(ctx, invocation)
		}
		// DO INVOKE
		ivkURL := ivk.GetURL().Key()
		res = ivk.Invoke(attemptCtx, invocation)
		if res.Error() != nil && !isBizError(res.Error()) {
			providers = append(providers, ivkURL)
			retryErr = res.Error()
			continue
		}
		return res
	}
	ip := common.GetLocalIp()
	invokerSvc := invoker.GetURL().Service()
	invokerUrl := invoker.Directory.GetURL()
	if ivk == nil {
		if observer := cluster.DiagnosticsFromContext(ctx, invocation); observer != nil {
			observer.NoProvider()
		}
		logger.CtxErrorf(ctx, "[Cluster][Failover] no provider available, method=%s service=%s", methodName, invokerSvc)
		return &result.RPCResult{
			Err: fmt.Errorf("failed to invoke the method %s of the service %s .No provider is available because can't connect server",
				methodName, invokerSvc),
		}
	}

	// The registry URL can hold credentials and the error text can hold remote
	// data, so the span gets the method, service and counts only. The log line
	// keeps the rest.
	spanErr := fmt.Errorf("[Cluster][Failover] invoke failed after retries, method=%s service=%s retries=%d providers=%d/%d",
		methodName, invokerSvc, retries, len(providers), len(invokers))
	logger.CtxErrorf(dubboLogger.WithSpanError(ctx, spanErr),
		"[Cluster][Failover] invoke failed after retries, method=%s service=%s retries=%d providers=%d/%d registry=%s consumer=%s version=%s lastErr=%v",
		methodName, invokerSvc, retries, len(providers), len(invokers), invokerUrl, ip, constant.Version, res.Error().Error())

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
