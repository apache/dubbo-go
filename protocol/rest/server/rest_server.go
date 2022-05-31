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

package server

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strconv"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	rconfig "dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

const parseParameterErrorStr = "An error occurred while parsing parameters on the server"

// RestServer user can implement this server interface
type RestServer interface {
	// Start rest server
	Start(url *common.URL)
	// Deploy a http api
	Deploy(restMethodConfig *rconfig.RestMethodConfig, routeFunc func(request RestServerRequest, response RestServerResponse))
	// UnDeploy a http api
	UnDeploy(restMethodConfig *rconfig.RestMethodConfig)
	// Destroy rest server
	Destroy()
}

// RestServerRequest interface
type RestServerRequest interface {
	// RawRequest get the Ptr of http.Request
	RawRequest() *http.Request
	// PathParameter get the path parameter by name
	PathParameter(name string) string
	// PathParameters get the map of the path parameters
	PathParameters() map[string]string
	// QueryParameter get the query parameter by name
	QueryParameter(name string) string
	// QueryParameters get the map of query parameters
	QueryParameters(name string) []string
	// BodyParameter get the body parameter of name
	BodyParameter(name string) (string, error)
	// HeaderParameter get the header parameter of name
	HeaderParameter(name string) string
	// ReadEntity checks the Accept header and reads the content into the entityPointer.
	ReadEntity(entityPointer interface{}) error
}

// RestServerResponse interface
type RestServerResponse interface {
	http.ResponseWriter
	// WriteError writes the http status and the error string on the response. err can be nil.
	// Return an error if writing was not successful.
	WriteError(httpStatus int, err error) (writeErr error)
	// WriteEntity marshals the value using the representation denoted by the Accept Header.
	WriteEntity(value interface{}) error
}

// GetRouteFunc is a route function will be invoked by http server
func GetRouteFunc(invoker protocol.Invoker, methodConfig *rconfig.RestMethodConfig) func(req RestServerRequest, resp RestServerResponse) {
	return func(req RestServerRequest, resp RestServerResponse) {
		var (
			err  error
			args []interface{}
		)
		svc := common.ServiceMap.GetServiceByServiceKey(invoker.GetURL().Protocol, invoker.GetURL().ServiceKey())
		// get method
		method := svc.Method()[methodConfig.MethodName]
		argsTypes := method.ArgsType()
		replyType := method.ReplyType()
		// two ways to prepare arguments
		// if method like this 'func1(req []interface{}, rsp *User) error'
		// we don't have arguments type
		if (len(argsTypes) == 1 || len(argsTypes) == 2 && replyType == nil) &&
			argsTypes[0].String() == "[]interface {}" {
			args, err = getArgsInterfaceFromRequest(req, methodConfig)
		} else {
			args, err = getArgsFromRequest(req, argsTypes, methodConfig)
		}
		if err != nil {
			logger.Errorf("[Go Restful] parsing http parameters error:%v", err)
			err = resp.WriteError(http.StatusInternalServerError, errors.New(parseParameterErrorStr))
			if err != nil {
				logger.Errorf("[Go Restful] WriteErrorString error:%v", err)
			}
		}
		result := invoker.Invoke(context.Background(), invocation.NewRPCInvocation(methodConfig.MethodName, args, make(map[string]interface{})))
		if result.Error() != nil {
			err = resp.WriteError(http.StatusInternalServerError, result.Error())
			if err != nil {
				logger.Errorf("[Go Restful] WriteError error:%v", err)
			}
			return
		}
		err = resp.WriteEntity(result.Result())
		if err != nil {
			logger.Errorf("[Go Restful] WriteEntity error:%v", err)
		}
	}
}

// getArgsInterfaceFromRequest when service function like GetUser(req []interface{}, rsp *User) error
// use this method to get arguments
func getArgsInterfaceFromRequest(req RestServerRequest, methodConfig *rconfig.RestMethodConfig) ([]interface{}, error) {
	argsMap := make(map[int]interface{}, 8)
	maxKey := 0
	for k, v := range methodConfig.PathParamsMap {
		if maxKey < k {
			maxKey = k
		}
		argsMap[k] = req.PathParameter(v)
	}
	for k, v := range methodConfig.QueryParamsMap {
		if maxKey < k {
			maxKey = k
		}
		params := req.QueryParameters(v)
		if len(params) == 1 {
			argsMap[k] = params[0]
		} else {
			argsMap[k] = params
		}
	}
	for k, v := range methodConfig.HeadersMap {
		if maxKey < k {
			maxKey = k
		}
		argsMap[k] = req.HeaderParameter(v)
	}
	if methodConfig.Body >= 0 {
		if maxKey < methodConfig.Body {
			maxKey = methodConfig.Body
		}
		m := make(map[string]interface{})

		if err := req.ReadEntity(&m); err != nil {

			//return nil, perrors.Errorf("[Go restful] Read body entity as map[string]interface{} error:%v", err)
			logger.Warnf("[Go Restful] parsing http parameters by body entity error: %v", err)
		} else {
			argsMap[methodConfig.Body] = m
		}
	}
	args := make([]interface{}, maxKey+1)
	for k, v := range argsMap {
		if k >= 0 {
			args[k] = v
		}
	}
	return args, nil
}

// getArgsFromRequest get arguments from server.RestServerRequest
func getArgsFromRequest(req RestServerRequest, argsTypes []reflect.Type, methodConfig *rconfig.RestMethodConfig) ([]interface{}, error) {
	argsLength := len(argsTypes)
	args := make([]interface{}, argsLength)
	for i, t := range argsTypes {
		args[i] = reflect.Zero(t).Interface()
	}
	if err := assembleArgsFromPathParams(methodConfig, argsLength, argsTypes, req, args); err != nil {
		return nil, err
	}
	if err := assembleArgsFromQueryParams(methodConfig, argsLength, argsTypes, req, args); err != nil {
		return nil, err
	}
	if err := assembleArgsFromBody(methodConfig, argsTypes, req, args); err != nil {
		return nil, err
	}
	if err := assembleArgsFromHeaders(methodConfig, req, argsLength, argsTypes, args); err != nil {
		return nil, err
	}
	return args, nil
}

// assembleArgsFromHeaders assemble arguments from headers
func assembleArgsFromHeaders(methodConfig *rconfig.RestMethodConfig, req RestServerRequest, argsLength int, argsTypes []reflect.Type, args []interface{}) error {
	for k, v := range methodConfig.HeadersMap {
		param := req.HeaderParameter(v)
		if k < 0 || k >= argsLength {
			return perrors.Errorf("[Go restful] Header param parse error, the index %v args of method:%v doesn't exist", k, methodConfig.MethodName)
		}
		t := argsTypes[k]
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.String {
			args[k] = param
		} else {
			return perrors.Errorf("[Go restful] Header param parse error, the index %v args's type isn't string", k)
		}
	}
	return nil
}

// assembleArgsFromBody assemble arguments from body
func assembleArgsFromBody(methodConfig *rconfig.RestMethodConfig, argsTypes []reflect.Type, req RestServerRequest, args []interface{}) error {
	if methodConfig.Body >= 0 && methodConfig.Body < len(argsTypes) {
		t := argsTypes[methodConfig.Body]
		kind := t.Kind()
		if kind == reflect.Ptr {
			t = t.Elem()
		}
		var ni interface{}
		if t.String() == "[]interface {}" {
			ni = make([]map[string]interface{}, 0)
		} else if t.String() == "interface {}" {
			ni = make(map[string]interface{})
		} else {
			n := reflect.New(t)
			if n.CanInterface() {
				ni = n.Interface()
			}
		}
		if err := req.ReadEntity(&ni); err != nil {

			//return perrors.Errorf("[Go restful] Read body entity error, error is %v", perrors.WithStack(err))
			logger.Warnf("[Go Restful] parsing http parameters by body entity error: %v", err)
		} else {
			args[methodConfig.Body] = ni
		}
	}
	return nil
}

// assembleArgsFromQueryParams assemble arguments from query params
func assembleArgsFromQueryParams(methodConfig *rconfig.RestMethodConfig, argsLength int, argsTypes []reflect.Type, req RestServerRequest, args []interface{}) error {
	var (
		err   error
		param interface{}
		i64   int64
	)
	for k, v := range methodConfig.QueryParamsMap {
		if k < 0 || k >= argsLength {
			return perrors.Errorf("[Go restful] Query param parse error, the index %v args of method:%v doesn't exist", k, methodConfig.MethodName)
		}
		t := argsTypes[k]
		kind := t.Kind()
		if kind == reflect.Ptr {
			t = t.Elem()
			kind = t.Kind()
		}
		if kind == reflect.Slice {
			param = req.QueryParameters(v)
		} else if kind == reflect.String {
			param = req.QueryParameter(v)
		} else if kind == reflect.Int {
			param, err = strconv.Atoi(req.QueryParameter(v))
		} else if kind == reflect.Int32 {
			i64, err = strconv.ParseInt(req.QueryParameter(v), 10, 32)
			if err == nil {
				param = int32(i64)
			}
		} else if kind == reflect.Int64 {
			param, err = strconv.ParseInt(req.QueryParameter(v), 10, 64)
		} else {
			return perrors.Errorf("[Go restful] Query param parse error, the index %v args's type isn't int or string or slice", k)
		}
		if err != nil {
			return perrors.Errorf("[Go restful] Query param parse error, error:%v", perrors.WithStack(err))
		}
		args[k] = param
	}
	return nil
}

// assembleArgsFromPathParams assemble arguments from path params
func assembleArgsFromPathParams(methodConfig *rconfig.RestMethodConfig, argsLength int, argsTypes []reflect.Type, req RestServerRequest, args []interface{}) error {
	var (
		err   error
		param interface{}
		i64   int64
	)
	for k, v := range methodConfig.PathParamsMap {
		if k < 0 || k >= argsLength {
			return perrors.Errorf("[Go restful] Path param parse error, the index %v args of method:%v doesn't exist", k, methodConfig.MethodName)
		}
		t := argsTypes[k]
		kind := t.Kind()
		if kind == reflect.Ptr {
			t = t.Elem()
			kind = t.Kind()
		}
		if kind == reflect.Int {
			param, err = strconv.Atoi(req.PathParameter(v))
		} else if kind == reflect.Int32 {
			i64, err = strconv.ParseInt(req.PathParameter(v), 10, 32)
			if err == nil {
				param = int32(i64)
			}
		} else if kind == reflect.Int64 {
			param, err = strconv.ParseInt(req.PathParameter(v), 10, 64)
		} else if kind == reflect.String {
			param = req.PathParameter(v)
		} else {
			return perrors.Errorf("[Go restful] Path param parse error, the index %v args's type isn't int or string", k)
		}
		if err != nil {
			return perrors.Errorf("[Go restful] Path param parse error, error is %v", perrors.WithStack(err))
		}
		args[k] = param
	}
	return nil
}
