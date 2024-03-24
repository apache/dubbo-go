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
	jwt2 "github.com/lestrrat-go/jwx/v2/jwt"
	"strings"
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
	if f.pilotAgent.GetHostInboundListener().JwtAuthnFilter.JwtAuthentication == nil {
		// there is no jwt authn filter
		return invoker.Invoke(ctx, invocation)
	}

	requestInfo := buildRequestInfoFromCtx(ctx, invoker, invocation)
	JwtAuthentication := f.pilotAgent.GetHostInboundListener().JwtAuthnFilter.JwtAuthentication
	providers := JwtAuthentication.Providers

	for _, rule := range JwtAuthentication.Rules {
		if matcheRule(requestInfo, rule) {
			requires := rule.Requires
			// get all provider names
			allProviderNames := requires.ProviderNames
			allowMissingOrFailed := requires.AllowMissing
			allowMissing := requires.AllowMissingOrFailed
			jwtToken, findProviderName, findHeadName, tokenExists, tokenVerified := verifyByProviderNames(allProviderNames, providers, requestInfo)
			if jwtToken == nil {
				if tokenVerified && !allowMissingOrFailed {
					// return fail result
				}
				if !tokenExists && !allowMissing {
					// return fail result
				}
			}
			// handle jwtToken
			findProvider := providers[findProviderName]
			if !findProvider.Forward {
				// clear token
				delete(invocation.Attachments(), findHeadName)
			}
			//

			break
		}
	}

	logger.Infof("jwt filter invoker:%v invocation:%v", invoker, invocation)
	return invoker.Invoke(ctx, invocation)
}

// OnResponse dummy process, returns the result directly
func (f *jwtAuthnFilter) OnResponse(_ context.Context, result protocol.Result, _ protocol.Invoker,
	_ protocol.Invocation) protocol.Result {
	return result
}

func matcheRule(requestInfo map[string]string, rule *resources.JwtRequirementRule) bool {
	path := requestInfo[":path"]
	return rule.Match.Match(path)
}

func extractJWTFromRequest(requestInfo map[string]string, fromHeaders []resources.JwtHeader) (string, string, error) {
	for _, header := range fromHeaders {
		name := header.Name
		valuePrefix := header.ValuePrefix
		name = strings.ToLower(name)
		if v, ok := requestInfo[name]; ok {
			// find token
			if len(valuePrefix) > 0 {
				if strings.HasPrefix(v, valuePrefix) {
					return strings.TrimPrefix(v, valuePrefix), name, nil
				} else {
					return "", "", fmt.Errorf("invalid authorization header format")
				}
			}
			return v, name, nil
		}
	}
	return "", "", nil
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

func verifyByProviderNames(allProviderNames []string, providers map[string]*resources.JwtProvider, requestInfo map[string]string) (jwt2.Token, string, string, bool, bool) {
	var (
		tokenExists   bool
		tokenVerified bool
	)
	findProviderName := ""
	findHeaderName := ""
	for _, providerName := range allProviderNames {
		provider, ok := providers[providerName]
		findProviderName = providerName
		if !ok {
			continue
		}
		if provider.LocalJwks.Keys == nil {
			continue
		}

		token, headName, tokenErr := extractJWTFromRequest(requestInfo, provider.FromHeaders)
		if tokenErr != nil {
			tokenExists = true
			continue
		}
		if len(token) == 0 {
			tokenExists = false
			continue
		}

		findHeaderName = headName
		tokenExists = true

		// verify token
		jwtToken, err := resources.ValidateAndParseJWT(token, provider.LocalJwks.Keys)
		if err != nil {
			tokenVerified = false
			continue
		}

		tokenVerified = true
		// todo check jwt issuer
		issuerVerified := true
		if len(provider.Issuer) > 0 {
			issuerVerified = false
			if jwtToken.Issuer() == provider.Issuer {
				issuerVerified = true
			}
		}
		if !issuerVerified {
			tokenVerified = false
			continue
		}

		// todo check jwt aud
		audVerfiyed := true
		if len(provider.Audiences) > 0 {
			audVerfiyed = false
			for _, aud := range provider.Audiences {
				for _, tAud := range jwtToken.Audience() {
					if aud == tAud {
						audVerfiyed = true
						break
					}
				}
			}
		}
		if !audVerfiyed {
			tokenVerified = false
			continue
		}

		tokenVerified = true

		if jwtToken != nil {
			return jwtToken, findProviderName, findHeaderName, tokenExists, tokenVerified
		}

		jwtToken.PrivateClaims()

	}
	return nil, findProviderName, findHeaderName, tokenExists, tokenVerified
}
