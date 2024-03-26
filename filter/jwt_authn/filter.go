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
	istiofilter "dubbo.apache.org/dubbo-go/v3/istio/filter"
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"encoding/base64"
	"errors"
	"fmt"
	"sync"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/istio"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"github.com/dubbogo/gost/log/logger"
)

var (
	once sync.Once
	jwt  *jwtAuthnFilter
)

func init() {
	extension.SetFilter(constant.JwtFilterKey, newJwtAuthnFilter)
}

type jwtAuthnFilter struct {
	pilotAgent istio.XdsAgent
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

	jwtAuthentication := f.pilotAgent.GetHostInboundJwtAuthentication()
	if jwtAuthentication == nil {
		// there is no jwt authn filter
		logger.Info("[jwt authn filter] skip jwt authn because there is no jwt authentication found.")
		return invoker.Invoke(ctx, invocation)
	}

	headers := buildRequestHeadersFromCtx(ctx, invoker, invocation)
	jwtAuthnFilterEngine := istiofilter.NewJwtAuthnFilterEngine(headers, jwtAuthentication)
	jwtAuthnResult, err := jwtAuthnFilterEngine.Filter()
	if err != nil {
		result := &protocol.RPCResult{}
		result.SetResult(nil)
		result.SetError(err)
		return result
	}

	jwtVerifyStatus := jwtAuthnResult.JwtVerfiyStatus
	logger.Infof("[jwt authn filter] final jwt verify status: %t", jwtVerifyStatus)
	// check final result
	if jwtVerifyStatus == istiofilter.JwtVerfiyStatusMissing {
		result := &protocol.RPCResult{}
		result.SetResult(nil)
		result.SetError(errors.New("jwt token is missing"))
		return result
	}

	if jwtVerifyStatus == istiofilter.JwtVerfiyStatusFailed {
		result := &protocol.RPCResult{}
		result.SetResult(nil)
		result.SetError(errors.New("jwt token verify fail"))
		return result
	}

	findProviderName := jwtAuthnResult.FindProviderName
	findHeaderName := jwtAuthnResult.FindHeaderName
	jwtToken := jwtAuthnResult.JwtToken
	providers := jwtAuthentication.Providers
	// handle jwtToken
	if len(findProviderName) > 0 && jwtToken != nil {
		findProvider := providers[findProviderName]
		if !findProvider.Forward {
			// clear token in attachments
			delete(invocation.Attachments(), findHeaderName)
		}

		if jwtJsonClaims, err := resources.ConvertJwtTokenToJwtClaimsJson(jwtToken); err != nil {
			logger.Errorf("[jwt authn filter]  can not convert from jwt token to jwt claims, err:%v", err)
		} else {
			if len(findProvider.ForwardPayloadHeader) > 0 {
				invocation.SetAttachment(findProvider.ForwardPayloadHeader, []string{base64.URLEncoding.EncodeToString([]byte(jwtJsonClaims))})
			}
			// add new attachment named x-jwt-claims
			logger.Infof("[jwt authn filter] add attachment k: %s, v: %s", constant.HttpHeaderXJwtClaimsName, jwtJsonClaims)
			invocation.SetAttachment(constant.HttpHeaderXJwtClaimsName, []string{jwtJsonClaims})
		}
	}

	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (f *jwtAuthnFilter) OnResponse(_ context.Context, result protocol.Result, _ protocol.Invoker,
	_ protocol.Invocation) protocol.Result {
	return result
}

func buildRequestHeadersFromCtx(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) map[string]string {
	return utils.ConvertAttachmentsToMap(invocation.Attachments())
}
