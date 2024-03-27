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
)

type MTLSResult struct {
	ReqOK  bool
	ReqMsg string
}

type MTLSFilterEngine struct {
	headers       map[string]string
	mutualTLSMode resources.MutualTLSMode
}

func NewMTLSFilterEngine(headers map[string]string, mutualTLSMode resources.MutualTLSMode) *MTLSFilterEngine {
	mtlsEngine := &MTLSFilterEngine{
		headers:       headers,
		mutualTLSMode: mutualTLSMode,
	}
	return mtlsEngine
}

func (m *MTLSFilterEngine) Filter() (*MTLSResult, error) {

	scheme := "https"
	if _, ok := m.headers[constant.HttpHeaderXSchemeName]; ok {
		scheme = strings.ToLower(m.headers[constant.HttpHeaderXSchemeName])
	}

	reqOKMsg := fmt.Sprintf("%s request on mtls %s mode is granted", scheme, resources.MutualTLSModeToString(m.mutualTLSMode))
	reqForbiddenMsg := fmt.Sprintf("%s request on mtls %s mode is forbidden", scheme, resources.MutualTLSModeToString(m.mutualTLSMode))

	reqOK := true
	reqMsg := reqOKMsg

	switch m.mutualTLSMode {
	case resources.MTLSUnknown:
		reqOK = false
		reqMsg = reqForbiddenMsg
	case resources.MTLSPermissive:
		reqOK = true
		reqMsg = reqOKMsg
	case resources.MTLSStrict:
		if scheme == "http" {
			reqOK = false
			reqMsg = reqForbiddenMsg
		}
	case resources.MTLSDisable:
		if scheme == "https" {
			reqOK = false
			reqMsg = reqForbiddenMsg
		}
	default:
	}

	mtlsResult := &MTLSResult{
		ReqOK:  reqOK,
		ReqMsg: reqMsg,
	}
	return mtlsResult, nil
}
