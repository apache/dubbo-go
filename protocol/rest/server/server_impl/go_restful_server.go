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

package server_impl

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/emicklei/go-restful/v3"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/rest/config"
	"github.com/apache/dubbo-go/protocol/rest/server"
)

func init() {
	extension.SetRestServer(constant.DEFAULT_REST_SERVER, GetNewGoRestfulServer)
}

var filterSlice []restful.FilterFunction

type GoRestfulServer struct {
	srv       *http.Server
	container *restful.Container
}

func NewGoRestfulServer() *GoRestfulServer {
	return &GoRestfulServer{}
}

func (grs *GoRestfulServer) Start(url common.URL) {
	grs.container = restful.NewContainer()
	for _, filter := range filterSlice {
		grs.container.Filter(filter)
	}
	grs.srv = &http.Server{
		Handler: grs.container,
	}
	ln, err := net.Listen("tcp", url.Location)
	if err != nil {
		panic(perrors.New(fmt.Sprintf("Restful Server start error:%v", err)))
	}

	go func() {
		err := grs.srv.Serve(ln)
		if err != nil && err != http.ErrServerClosed {
			logger.Errorf("[Go Restful] http.server.Serve(addr{%s}) = err{%+v}", url.Location, err)
		}
	}()
}

func (grs *GoRestfulServer) Deploy(invoker protocol.Invoker, restServiceConfig *config.RestServiceConfig) {
	ws := &restful.WebService{}
	ws.Path(restServiceConfig.Path)

	svc := common.ServiceMap.GetService(invoker.GetUrl().Protocol, strings.TrimPrefix(invoker.GetUrl().Path, "/"))

	for methodName, config := range restServiceConfig.RestMethodConfigsMap {
		// get method
		method := svc.Method()[methodName]
		argsTypes := method.ArgsType()
		replyType := method.ReplyType()
		ws.Route(
			ws.Method(config.MethodType).
				Path(config.Path).
				Produces(strings.Split(config.Produces, ",")...).
				Consumes(strings.Split(config.Consumes, ",")...).
				To(getFunc(methodName, invoker, argsTypes, replyType, config)))
	}
	grs.container.Add(ws)
}

func getFunc(methodName string, invoker protocol.Invoker, argsTypes []reflect.Type,
	replyType reflect.Type, config *config.RestMethodConfig) func(req *restful.Request, resp *restful.Response) {
	return func(req *restful.Request, resp *restful.Response) {
		var (
			err  error
			args []interface{}
		)
		if (len(argsTypes) == 1 || len(argsTypes) == 2 && replyType == nil) &&
			argsTypes[0].String() == "[]interface {}" {
			args = getArgsInterfaceFromRequest(req, config)
		} else {
			args = getArgsFromRequest(req, argsTypes, config)
		}
		result := invoker.Invoke(context.Background(), invocation.NewRPCInvocation(methodName, args, make(map[string]string)))
		if result.Error() != nil {
			err = resp.WriteError(http.StatusInternalServerError, result.Error())
			if err != nil {
				logger.Errorf("[Go Restful] WriteError error:%v", err)
			}
			return
		}
		err = resp.WriteEntity(result.Result())
		if err != nil {
			logger.Error("[Go Restful] WriteEntity error:%v", err)
		}
	}
}
func (grs *GoRestfulServer) UnDeploy(restServiceConfig *config.RestServiceConfig) {
	ws := new(restful.WebService)
	ws.Path(restServiceConfig.Path)
	err := grs.container.Remove(ws)
	if err != nil {
		logger.Warnf("[Go restful] Remove web service error:%v", err)
	}
}

func (grs *GoRestfulServer) Destroy() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := grs.srv.Shutdown(ctx); err != nil {
		logger.Errorf("[Go Restful] Server Shutdown:", err)
	}
	logger.Infof("[Go Restful] Server exiting")
}

func getArgsInterfaceFromRequest(req *restful.Request, config *config.RestMethodConfig) []interface{} {
	argsMap := make(map[int]interface{}, 8)
	maxKey := 0
	for k, v := range config.PathParamsMap {
		if maxKey < k {
			maxKey = k
		}
		argsMap[k] = req.PathParameter(v)
	}
	for k, v := range config.QueryParamsMap {
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
	for k, v := range config.HeadersMap {
		if maxKey < k {
			maxKey = k
		}
		argsMap[k] = req.HeaderParameter(v)
	}
	if config.Body >= 0 {
		if maxKey < config.Body {
			maxKey = config.Body
		}
		m := make(map[string]interface{})
		// TODO read as a slice
		if err := req.ReadEntity(&m); err != nil {
			logger.Warnf("[Go restful] Read body entity as map[string]interface{} error:%v", perrors.WithStack(err))
		} else {
			argsMap[config.Body] = m
		}
	}
	args := make([]interface{}, maxKey+1)
	for k, v := range argsMap {
		if k >= 0 {
			args[k] = v
		}
	}
	return args
}

func getArgsFromRequest(req *restful.Request, argsTypes []reflect.Type, config *config.RestMethodConfig) []interface{} {
	argsLength := len(argsTypes)
	args := make([]interface{}, argsLength)
	for i, t := range argsTypes {
		args[i] = reflect.Zero(t).Interface()
	}
	var (
		err   error
		param interface{}
		i64   int64
	)
	for k, v := range config.PathParamsMap {
		if k < 0 || k >= argsLength {
			logger.Errorf("[Go restful] Path param parse error, the args:%v doesn't exist", k)
			continue
		}
		t := argsTypes[k]
		kind := t.Kind()
		if kind == reflect.Ptr {
			t = t.Elem()
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
			logger.Warnf("[Go restful] Path param parse error, the args:%v of type isn't int or string", k)
			continue
		}
		if err != nil {
			logger.Errorf("[Go restful] Path param parse error, error is %v", err)
			continue
		}
		args[k] = param
	}
	for k, v := range config.QueryParamsMap {
		if k < 0 || k >= argsLength {
			logger.Errorf("[Go restful] Query param parse error, the args:%v doesn't exist", k)
			continue
		}
		t := argsTypes[k]
		kind := t.Kind()
		if kind == reflect.Ptr {
			t = t.Elem()
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
			logger.Errorf("[Go restful] Query param parse error, the args:%v of type isn't int or string or slice", k)
			continue
		}
		if err != nil {
			logger.Errorf("[Go restful] Query param parse error, error is %v", err)
			continue
		}
		args[k] = param
	}

	if config.Body >= 0 && config.Body < len(argsTypes) {
		t := argsTypes[config.Body]
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
			logger.Errorf("[Go restful] Read body entity error:%v", err)
		} else {
			args[config.Body] = ni
		}
	}

	for k, v := range config.HeadersMap {
		param := req.HeaderParameter(v)
		if k < 0 || k >= argsLength {
			logger.Errorf("[Go restful] Header param parse error, the args:%v doesn't exist", k)
			continue
		}
		t := argsTypes[k]
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() == reflect.String {
			args[k] = param
		} else {
			logger.Errorf("[Go restful] Header param parse error, the args:%v of type isn't string", k)
		}
	}

	return args
}

func GetNewGoRestfulServer() server.RestServer {
	return NewGoRestfulServer()
}

// Let user addFilter
// addFilter should before config.Load()
func AddGoRestfulServerFilter(filterFuc restful.FilterFunction) {
	filterSlice = append(filterSlice, filterFuc)
}
