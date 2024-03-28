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

package filter

import (
	"fmt"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"github.com/dubbogo/gost/log/logger"
	jwt2 "github.com/lestrrat-go/jwx/v2/jwt"
)

type JwtVerfiyStatus int32

const (
	JwtVerfiyStatusOK JwtVerfiyStatus = iota
	JwtVerfiyStatusMissing
	JwtVerfiyStatusFailed
)

type JwtAuthnResult struct {
	JwtVerfiyStatus  JwtVerfiyStatus
	TokenExists      bool
	TokenVerified    bool
	JwtToken         jwt2.Token
	FindProviderName string
	FindHeaderName   string
	FindToken        string
}

type JwtAuthnFilterEngine struct {
	headers        map[string]string
	authentication *resources.JwtAuthentication
}

func NewJwtAuthnFilterEngine(headers map[string]string, authentication *resources.JwtAuthentication) *JwtAuthnFilterEngine {
	jwtAuthnFilterEngine := &JwtAuthnFilterEngine{
		headers:        headers,
		authentication: authentication,
	}
	return jwtAuthnFilterEngine
}

func (e *JwtAuthnFilterEngine) Filter() (*JwtAuthnResult, error) {
	logger.Infof("[jwt authn filter] authentication: %s", utils.ConvertJsonString(e.authentication))
	providers := e.authentication.Providers
	path := e.headers[constant.HttpHeaderXPathName]
	var (
		jwtToken                                    jwt2.Token
		findProviderName, findHeaderName, findToken string
		tokenExists, tokenVerified                  bool
	)
	jwtVerifyStatus := JwtVerfiyStatusOK

	for _, rule := range e.authentication.Rules {
		if rule.Match.Match(path) {
			logger.Infof("[jwt authn filter] match path :%s on rule by rule match action :%s and rule value :%s", path, rule.Match.Action, rule.Match.Value)
			logger.Infof("[jwt authn filter] match rule info:%s", utils.ConvertJsonString(rule))
			requires := rule.Requires
			// get all provider names
			allProviderNames := requires.ProviderNames
			allowMissingOrFailed := requires.AllowMissingOrFailed
			allowMissing := requires.AllowMissing
			jwtToken, findProviderName, findHeaderName, findToken, tokenExists, tokenVerified = verifyByProviderNames(allProviderNames, providers, e.headers)
			logger.Infof("[jwt authn filter] match result: provider name: %s, header name: %s, tokenExists: %t, tokenVerified :%t", findProviderName, findHeaderName, tokenExists, tokenVerified)
			logger.Infof("[jwt authn filter] match result: found token: %s", findToken)
			if jwtToken == nil {
				if !tokenExists && !allowMissing {
					jwtVerifyStatus = JwtVerfiyStatusMissing
				}
				if tokenExists && !tokenVerified && !allowMissingOrFailed {
					jwtVerifyStatus = JwtVerfiyStatusFailed
				}
			}
			break
		}
	}

	jwtAuthnResult := &JwtAuthnResult{
		JwtToken:         jwtToken,
		FindProviderName: findProviderName,
		FindToken:        findToken,
		FindHeaderName:   findHeaderName,
		JwtVerfiyStatus:  jwtVerifyStatus,
		TokenExists:      tokenExists,
		TokenVerified:    tokenVerified,
	}
	logger.Infof("[jwt authn filter] final jwt verify result: %s", utils.ConvertJsonString(jwtAuthnResult))
	return jwtAuthnResult, nil
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

func verifyByProviderNames(allProviderNames []string, providers map[string]*resources.JwtProvider, requestInfo map[string]string) (jwt2.Token, string, string, string, bool, bool) {
	var (
		tokenExists   bool
		tokenVerified bool
	)
	findProviderName := ""
	findHeaderName := ""
	findToken := ""

	for _, providerName := range allProviderNames {
		// get provider from name
		provider, ok := providers[providerName]
		findProviderName = providerName
		if !ok {
			continue
		}
		if provider.LocalJwks.Keys == nil {
			continue
		}

		// extract token from headers
		tokenExists = false
		token, headName, tokenErr := extractJWTFromRequest(requestInfo, provider.FromHeaders)
		if tokenErr != nil {
			tokenExists = true
			continue
		}
		if len(token) == 0 {
			tokenExists = false
			continue
		}
		tokenExists = true
		findToken = token
		findHeaderName = headName

		// verify token and return jwt token
		tokenVerified = false
		jwtToken, err := resources.ValidateAndParseJWT(token, provider.LocalJwks.Keys)
		if err != nil {
			tokenVerified = false
			continue
		}
		// verify token by issuer
		issuerVerified := resources.ValidateJwtTokenByIssuer(provider.Issuer, jwtToken)
		if !issuerVerified {
			tokenVerified = false
			continue
		}
		// verify token by audiences
		audVerfiyed := resources.ValidateJwtTokenByAudiences(provider.Audiences, jwtToken)
		if !audVerfiyed {
			tokenVerified = false
			continue
		}
		tokenVerified = true

		// return token which is verified
		return jwtToken, findProviderName, findHeaderName, findToken, tokenExists, tokenVerified
	}
	return nil, findProviderName, findHeaderName, findToken, tokenExists, tokenVerified
}
