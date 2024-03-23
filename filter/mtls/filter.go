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
	"dubbo.apache.org/dubbo-go/v3/istio"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	once sync.Once
	mtls *mtlsFilter
)

func init() {
	extension.SetFilter(constant.MtlsFilterKey, newMtlsFilter)
}

type mtlsFilter struct {
	pilotAgent    *istio.PilotAgent
	mutualTLSMode resources.MutualTLSMode
}

func newMtlsFilter() filter.Filter {
	if mtls == nil {
		once.Do(func() {
			pilotAgent, err := istio.GetPilotAgent(istio.PilotAgentTypeServerWorkload)
			if err != nil {
				logger.Errorf("[mtls filter] can not get pilot agent")
			}
			mtls = &mtlsFilter{
				pilotAgent:    pilotAgent,
				mutualTLSMode: resources.MTLSUnknown,
			}
		})
	}
	return mtls
}

// Invoke response to the callers with its first argument.
func (f *mtlsFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	// can not get if the current request is HTTPS or HTTP.
	// get request schema

	for key, attachment := range invocation.Attachments() {
		logger.Infof("get invocation attachment key %s = %+v", key, attachment)
	}

	attachments := ctx.Value(constant.AttachmentKey).(map[string]interface{})
	for key, attachment := range attachments {
		logger.Infof("get triple attachment key %s = %s", key, attachment.([]string)[0])
	}

	logger.Infof("get url %s", invoker.GetURL().String())
	logger.Infof("get path %s", fmt.Sprintf("%s/%s", invoker.GetURL().Path, invocation.MethodName()))
	// get newest mutualTLSMode
	mutualTLSMode := resources.MTLSUnknown
	reqOK := true
	reqMsg := ""
	if f.pilotAgent == nil {
		reqOK = false
		reqMsg = "pilot agent is not ready"
	} else {
		mutualTLSMode = f.pilotAgent.GetHostInboundMutualTLSMode()
	}

	f.mutualTLSMode = mutualTLSMode

	switch mutualTLSMode {
	case resources.MTLSUnknown:
	case resources.MTLSPermissive:
	case resources.MTLSStrict:
	case resources.MTLSDisable:
	default:
	}
	logger.Infof("get mutualTLSMode: %d ,reqOk:%t, reqMsg:%s", mutualTLSMode, reqOK, reqMsg)
	if !reqOK {
		return &protocol.RPCResult{
			Rest:  invocation.Arguments()[0],
			Attrs: invocation.Attachments(),
		}
	}
	// TODO it just now echo logger, and implement mtls support check later
	logger.Infof("mtls filter invoker:%v invocation:%v", invoker, invocation)
	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (f *mtlsFilter) OnResponse(ctx context.Context, result protocol.Result, _ protocol.Invoker,
	invocation protocol.Invocation) protocol.Result {
	invocation.SetAttachment("mutual-tls-mode", []string{resources.MutualTLSModeToString(f.mutualTLSMode)})
	return result
}
