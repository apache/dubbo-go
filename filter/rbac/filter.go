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

package rbac

import (
	"context"
	"fmt"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/istio"
	istioengine "dubbo.apache.org/dubbo-go/v3/istio/engine"
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/dubbogo/gost/log/logger"
)

var (
	once sync.Once
	rbac *rbacFilter
)

func init() {
	extension.SetFilter(constant.RBACFilterKey, newRBACFilter)
}

type rbacFilter struct {
	pilotAgent istio.XdsAgent
}

func newRBACFilter() filter.Filter {
	if rbac == nil {
		once.Do(func() {
			pilotAgent, err := istio.GetPilotAgent(istio.PilotAgentTypeServerWorkload)
			if err != nil {
				logger.Errorf("[rbac filter] can't get pilot agent err:%v", err)
			}
			rbac = &rbacFilter{
				pilotAgent: pilotAgent,
			}
		})
	}
	return rbac
}

// Invoke processes the request and returns the result based on RBAC configuration.
func (f *rbacFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	logger.Infof("[rbac filter] invoker")
	if f.pilotAgent == nil {
		return &protocol.RPCResult{Err: fmt.Errorf("can not get pilot agent")}
	}
	if f.pilotAgent.GetHostInboundListener() == nil {
		return &protocol.RPCResult{Err: fmt.Errorf("can not get HostInboundListener in pilot agent")}
	}

	v3RBAC := f.pilotAgent.GetHostInboundRBAC()
	if v3RBAC == nil {
		// there is no jwt authn filter
		logger.Info("[rbac filter] skip rbac filter because there is no rbac configuration found.")
		return invoker.Invoke(ctx, invocation)
	}

	headers := buildRequestHeadersFromCtx(ctx, invoker, invocation)
	for key, attachment := range headers {
		logger.Infof("[rbac filter] invocation attachment key %s = %s", key, attachment)
	}
	rbacFilterEngine := istioengine.NewRBACFilterEngine(v3RBAC)
	rbacResult, err := rbacFilterEngine.Filter(headers)
	if err != nil {
		result := &protocol.RPCResult{}
		result.SetResult(nil)
		result.SetError(err)
		return result
	}

	if !rbacResult.ReqOK {
		result := &protocol.RPCResult{}
		result.SetResult(nil)
		result.SetError(fmt.Errorf("request deny by rbac filter policy: %s", rbacResult.MatchPolicyName))
		return result
	}

	logger.Infof("[rbac filter] rbac result: %s", utils.ConvertJsonString(rbacResult))
	return invoker.Invoke(ctx, invocation)
}

func buildRequestHeadersFromCtx(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) map[string]string {
	return utils.ConvertAttachmentsToMap(invocation.Attachments())
}

// OnResponse dummy process, returns the result directly
func (f *rbacFilter) OnResponse(_ context.Context, result protocol.Result, _ protocol.Invoker,
	_ protocol.Invocation) protocol.Result {

	return result
}
