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

package mtls

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/istio"
	istioengine "dubbo.apache.org/dubbo-go/v3/istio/engine"
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
	extension.SetFilter(constant.MTLSFilterKey, newMTLSFilter)
}

type mtlsFilter struct {
	pilotAgent istio.XdsAgent
}

func newMTLSFilter() filter.Filter {
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

// Invoke processes the request based on mTLS mode and returns the result.
// If the request fails the mTLS filtering, it returns an error result.
// Otherwise, it sets the corresponding mTLS mode attachment and invokes the next Invoker to handle the request.
func (f *mtlsFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("[mtls filter] url: %s", invoker.GetURL().String())
	headers := f.buildHeadersFromCtx(ctx, invoker, invocation)
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
	mtlsFilterEngine := istioengine.NewMTLSFilterEngine(mutualTLSMode)
	mtlsResult, _ := mtlsFilterEngine.Filter(headers)
	logger.Infof("[mtls filter] %s", mtlsResult.ReqMsg)

	if !mtlsResult.ReqOK {
		result := &protocol.RPCResult{}
		result.SetResult(nil)
		result.SetError(errors.New(mtlsResult.ReqMsg))
		return result
	}

	invocation.SetAttachment(constant.HttpHeaderXMTLSMode, []string{resources.MutualTLSModeToString(mutualTLSMode)})
	invocation.SetAttachment(constant.HttpHeaderXSourcePrincipal, []string{headers[constant.HttpHeaderXSpiffeName]})
	return invoker.Invoke(ctx, invocation)
}

func (f *mtlsFilter) buildHeadersFromCtx(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) map[string]string {
	headers := utils.ConvertAttachmentsToMap(invocation.Attachments())
	// build :path
	xpath := headers[constant.HttpHeaderXPathName]
	if len(xpath) == 0 {
		xpath = fmt.Sprintf("/%s/%s", invoker.GetURL().GetParam(constant.InterfaceKey, ""), invocation.MethodName())
		headers[constant.HttpHeaderXPathName] = xpath
		invocation.SetAttachment(constant.HttpHeaderXPathName, []string{xpath})
	}
	// build :scheme
	xscheme := headers[constant.HttpHeaderXSchemeName]
	if len(xscheme) == 0 {
		xscheme = "http"
		headers[constant.HttpHeaderXSchemeName] = xscheme
		invocation.SetAttachment(constant.HttpHeaderXSchemeName, []string{xscheme})
	}
	// build :method
	xmethod := headers[constant.HttpHeaderXMethodName]
	if len(xmethod) == 0 {
		xmethod = "POST"
		headers[constant.HttpHeaderXMethodName] = xmethod
		invocation.SetAttachment(constant.HttpHeaderXMethodName, []string{xmethod})
	}
	return headers
}

// OnResponse dummy process, returns the result directly
func (f *mtlsFilter) OnResponse(ctx context.Context, result protocol.Result, _ protocol.Invoker,
	invocation protocol.Invocation) protocol.Result {
	return result
}
