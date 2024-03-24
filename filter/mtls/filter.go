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
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"errors"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	"strings"
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
	pilotAgent *istio.PilotAgent
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
	// can not get if the current request is HTTPS or HTTP.
	// get request schema
	logger.Infof("[mtls filter] url: %s", invoker.GetURL().String())
	attachments := utils.ConvertAttachmentsToMap(invocation.Attachments())
	for key, attachment := range attachments {
		logger.Infof("[mtls filter] invocation attachment key %s = %s", key, attachment)
	}
	scheme := "https"
	if _, ok := attachments[":x-scheme"]; ok {
		scheme = strings.ToLower(attachments[":x-scheme"])
	}

	// get newest mutualTLSMode
	mutualTLSMode := resources.MTLSUnknown
	reqOK := true
	reqMsg := ""
	if f.pilotAgent == nil {
		reqOK = false
		reqMsg = "pilot agent is not ready"
	} else {
		reqOK = true
		mutualTLSMode = f.pilotAgent.GetHostInboundMutualTLSMode()
		reqMsg = fmt.Sprintf("%s request on mtls %s mode is granted", scheme, resources.MutualTLSModeToString(mutualTLSMode))
	}

	switch mutualTLSMode {
	case resources.MTLSUnknown:
		reqOK = false
		reqMsg = "request on mtls unknown mode is forbidden"
	case resources.MTLSPermissive:
		reqOK = true
	case resources.MTLSStrict:
		if scheme == "http" {
			reqOK = false
			reqMsg = "http request on mtls strict mode is forbidden"
		}
	case resources.MTLSDisable:
		if scheme == "https" {
			reqOK = false
			reqMsg = "https request on mtls disable mode is forbidden"
		}
	default:
	}
	logger.Infof("[mtls filter] scheme:%s, mutualTLSMode:%s ,reqOk:%t, reqMsg:%s", scheme, resources.MutualTLSModeToString(mutualTLSMode), reqOK, reqMsg)
	if !reqOK {
		result := &protocol.RPCResult{}
		result.SetResult(nil)
		result.SetError(errors.New(reqMsg))
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
