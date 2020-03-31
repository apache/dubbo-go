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
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	invocation_impl "github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/rest/client"
	"github.com/apache/dubbo-go/protocol/rest/config"
)

type RestInvoker struct {
	protocol.BaseInvoker
	client              client.RestClient
	restMethodConfigMap map[string]*config.RestMethodConfig
}

func NewRestInvoker(url common.URL, client *client.RestClient, restMethodConfig map[string]*config.RestMethodConfig) *RestInvoker {
	return &RestInvoker{
		BaseInvoker:         *protocol.NewBaseInvoker(url),
		client:              *client,
		restMethodConfigMap: restMethodConfig,
	}
}

func (ri *RestInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	inv := invocation.(*invocation_impl.RPCInvocation)
	methodConfig := ri.restMethodConfigMap[inv.MethodName()]
	var (
		result      protocol.RPCResult
		body        interface{}
		pathParams  map[string]string
		queryParams map[string]string
		headers     map[string]string
		err         error
	)
	if methodConfig == nil {
		result.Err = perrors.Errorf("[RestInvoker] Rest methodConfig:%s is nil", inv.MethodName())
		return &result
	}
	if pathParams, err = restStringMapTransform(methodConfig.PathParamsMap, inv.Arguments()); err != nil {
		result.Err = err
		return &result
	}
	if queryParams, err = restStringMapTransform(methodConfig.QueryParamsMap, inv.Arguments()); err != nil {
		result.Err = err
		return &result
	}
	if headers, err = restStringMapTransform(methodConfig.HeadersMap, inv.Arguments()); err != nil {
		result.Err = err
		return &result
	}
	if len(inv.Arguments()) > methodConfig.Body && methodConfig.Body >= 0 {
		body = inv.Arguments()[methodConfig.Body]
	}
	req := &client.RestClientRequest{
		Location:    ri.GetUrl().Location,
		Produces:    methodConfig.Produces,
		Consumes:    methodConfig.Consumes,
		Method:      methodConfig.MethodType,
		Path:        methodConfig.Path,
		PathParams:  pathParams,
		QueryParams: queryParams,
		Body:        body,
		Headers:     headers,
	}
	result.Err = ri.client.Do(req, inv.Reply())
	if result.Err == nil {
		result.Rest = inv.Reply()
	}
	return &result
}

func restStringMapTransform(paramsMap map[int]string, args []interface{}) (map[string]string, error) {
	resMap := make(map[string]string, len(paramsMap))
	for k, v := range paramsMap {
		if k >= len(args) || k < 0 {
			return nil, perrors.Errorf("[Rest Invoke] Index %v is out of bundle", k)
		}
		resMap[v] = fmt.Sprint(args[k])
	}
	return resMap, nil
}
