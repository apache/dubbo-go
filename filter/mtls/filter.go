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

// Package echo providers health check filter.
// RPCService need a Echo method in consumer, if you want to use Filter
// eg: Echo func(ctx context.Context, arg interface{}, rsp *Xxx) error
package echo

import (
	"context"
	"errors"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/istio"
	istiofilter "dubbo.apache.org/dubbo-go/v3/istio/filter"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/dubbogo/gost/log/logger"
)

var (
	once sync.Once
	mtls *mtlsFilter
)

func init() {
	extension.SetFilter(constant.MtlsFilterKey, newMtlsFilter)
}

type mtlsFilter struct {
	pilotAgent istio.XdsAgent
}

func newMtlsFilter() filter.Filter {
	if mtls == nil {
		once.Do(func() {
			pilotAgent, err := istio.GetPilotAgent(istio.PilotAgentTypeServerWorkload)
			if err != nil {
				logger.Errorf("[mtls filter] can not get pilot agent")
			}
			mtls = &mtlsFilter{
				pilotAgent: pilotAgent,
			}
		})
	}
	return mtls
}

// Invoke response to the callers with its first argument.
func (f *mtlsFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("[mtls filter] url: %s", invoker.GetURL().String())
	headers := utils.ConvertAttachmentsToMap(invocation.Attachments())
	for key, attachment := range headers {
		logger.Infof("[mtls filter] invocation attachment key %s = %s", key, attachment)
	}

	if f.pilotAgent == nil {
		result := &protocol.RPCResult{}
		result.SetResult(nil)
		result.SetError(errors.New("pilot agent is not ready"))
		return result
	}
	// filer request
	mutualTLSMode := f.pilotAgent.GetHostInboundMutualTLSMode()
	mtlsFilterEngine := istiofilter.NewMTLSFilterEngine(headers, mutualTLSMode)
	mtlsResult, _ := mtlsFilterEngine.Filter()
	logger.Infof("[mtls filter] %s", mtlsResult.ReqMsg)

	if !mtlsResult.ReqOK {
		result := &protocol.RPCResult{}
		result.SetResult(nil)
		result.SetError(errors.New(mtlsResult.ReqMsg))
		return result
	}

	invocation.SetAttachment(":mutual-tls-mode", []string{resources.MutualTLSModeToString(mutualTLSMode)})
	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (f *mtlsFilter) OnResponse(ctx context.Context, result protocol.Result, _ protocol.Invoker,
	invocation protocol.Invocation) protocol.Result {
	return result
}
