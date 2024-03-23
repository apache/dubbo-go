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
	"fmt"
	"strings"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/istio"
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	once sync.Once
	jwt  *jwtAuthnFilter
)

func init() {
	extension.SetFilter(constant.JwtFilterKey, newJwtAuthnFilter)
}

type jwtAuthnFilter struct {
	pilotAgent *istio.PilotAgent
}

func newJwtAuthnFilter() filter.Filter {
	if jwt == nil {
		once.Do(func() {
			pilotAgent, err := istio.GetPilotAgent(istio.PilotAgentTypeServerWorkload)
			if err != nil {
				logger.Errorf("[jwt authn can't get pilot agent err:%v", err)
			}
			jwt = &jwtAuthnFilter{
				pilotAgent: pilotAgent,
			}
		})
	}
	return jwt
}

// Invoke response to the callers with its first argument.
// Refer https://www.envoyproxy.io/docs/envoy/latest/api-v3/extensions/filters/http/jwt_authn/v3/config.proto.html#envoy-v3-api-msg-extensions-filters-http-jwt-authn-v3-jwtheader
func (f *jwtAuthnFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if f.pilotAgent == nil {
		return &protocol.RPCResult{Err: fmt.Errorf("can not get pilot agent")}
	}
	if f.pilotAgent.GetHostInboundListener() == nil {
		return &protocol.RPCResult{Err: fmt.Errorf("can not get HostInboundListener in pilot agent")}
	}
	logger.Infof("jwt filter invoker:%v invocation:%v", invoker, invocation)
	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (f *jwtAuthnFilter) OnResponse(_ context.Context, result protocol.Result, _ protocol.Invoker,
	_ protocol.Invocation) protocol.Result {
	return result
}

//func matcheRule(requestInfo map[string]string, rule rule) bool {
//
//}
//
//func isValidToken(token string, providerName string, config *JwtAuthentication) bool {
//
//}

func extractJWTFromRequest(requestInfo map[string]string, name string, valuePrefix string) (string, error) {
	name = strings.ToLower(name)
	if v, ok := requestInfo[name]; ok {
		// find token
		if len(valuePrefix) > 0 {
			if strings.HasPrefix(v, valuePrefix) {
				return strings.TrimPrefix(v, valuePrefix), nil
			} else {
				return "", fmt.Errorf("invalid authorization header format")
			}
		}
		return v, nil
	}
	return "", nil
}

func buildRequestInfoFromCtx(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) map[string]string {
	// get headers
	attachments := ctx.Value(constant.AttachmentKey).(map[string]interface{})
	requestInfo := make(map[string]string, 0)
	for key, attachment := range attachments {
		requestInfo[key] = attachment.([]string)[0]
	}
	// get request path
	requestInfo[":path"] = fmt.Sprintf("%s/%s", invoker.GetURL().Path, invocation.MethodName())
	return requestInfo
}
