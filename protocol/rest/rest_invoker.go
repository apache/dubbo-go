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

package rest

import (
	"context"
	"fmt"
	"net/http"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/rest/client"
	"dubbo.apache.org/dubbo-go/v3/protocol/rest/config"
	"dubbo.apache.org/dubbo-go/v3/protocol/result"
)

// nolint
type RestInvoker struct {
	base.BaseInvoker
	client              client.RestClient
	restMethodConfigMap map[string]*config.RestMethodConfig
}

// NewRestInvoker returns a RestInvoker
func NewRestInvoker(url *common.URL, client *client.RestClient, restMethodConfig map[string]*config.RestMethodConfig) *RestInvoker {
	return &RestInvoker{
		BaseInvoker:         *base.NewBaseInvoker(url),
		client:              *client,
		restMethodConfigMap: restMethodConfig,
	}
}

// Invoke is used to call service method by invocation
func (ri *RestInvoker) Invoke(ctx context.Context, inv base.Invocation) result.Result {
	rpcInv := inv.(*invocation.RPCInvocation)
	methodConfig := ri.restMethodConfigMap[rpcInv.MethodName()]
	var (
		result      result.RPCResult
		body        any
		pathParams  map[string]string
		queryParams map[string]string
		header      http.Header
		err         error
	)
	if methodConfig == nil {
		result.Err = perrors.Errorf("[RestInvoker] Rest methodConfig:%s is nil", rpcInv.MethodName())
		return &result
	}
	if pathParams, err = restStringMapTransform(methodConfig.PathParamsMap, rpcInv.Arguments()); err != nil {
		result.Err = err
		return &result
	}
	if queryParams, err = restStringMapTransform(methodConfig.QueryParamsMap, rpcInv.Arguments()); err != nil {
		result.Err = err
		return &result
	}
	if header, err = getRestHttpHeader(methodConfig, rpcInv.Arguments()); err != nil {
		result.Err = err
		return &result
	}
	if len(rpcInv.Arguments()) > methodConfig.Body && methodConfig.Body >= 0 {
		body = rpcInv.Arguments()[methodConfig.Body]
	}
	req := &client.RestClientRequest{
		Location:    ri.GetURL().Location,
		Method:      methodConfig.MethodType,
		Path:        methodConfig.Path,
		PathParams:  pathParams,
		QueryParams: queryParams,
		Body:        body,
		Header:      header,
	}
	result.Err = ri.client.Do(req, rpcInv.Reply())
	if result.Err == nil {
		result.Rest = rpcInv.Reply()
	}
	return &result
}

// restStringMapTransform is used to transform rest map
func restStringMapTransform(paramsMap map[int]string, args []any) (map[string]string, error) {
	resMap := make(map[string]string, len(paramsMap))
	for k, v := range paramsMap {
		if k >= len(args) || k < 0 {
			return nil, perrors.Errorf("[Rest Invoke] Index %v is out of bundle", k)
		}
		resMap[v] = fmt.Sprint(args[k])
	}
	return resMap, nil
}

// nolint
func getRestHttpHeader(methodConfig *config.RestMethodConfig, args []any) (http.Header, error) {
	header := http.Header{}
	headersMap := methodConfig.HeadersMap
	header.Set("Content-Type", methodConfig.Consumes)
	header.Set("Accept", methodConfig.Produces)
	for k, v := range headersMap {
		if k >= len(args) || k < 0 {
			return nil, perrors.Errorf("[Rest Invoke] Index %v is out of bundle", k)
		}
		header.Set(v, fmt.Sprint(args[k]))
	}
	return header, nil
}
