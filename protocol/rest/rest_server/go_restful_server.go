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

package rest_server

import (
	"context"
	"fmt"
	perrors "github.com/pkg/errors"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/rest/rest_interface"
	"github.com/emicklei/go-restful/v3"
)

func init() {
	extension.SetRestServer(constant.DEFAULT_REST_SERVER, GetNewGoRestfulServer)
}

type GoRestfulServer struct {
	srv       *http.Server
	container *restful.Container
}

func NewGoRestfulServer() *GoRestfulServer {
	return &GoRestfulServer{}
}

func (grs *GoRestfulServer) Start(url common.URL) {
	grs.container = restful.NewContainer()
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

func (grs *GoRestfulServer) Deploy(invoker protocol.Invoker, restMethodConfig map[string]*rest_interface.RestMethodConfig) {
	svc := common.ServiceMap.GetService(invoker.GetUrl().Protocol, strings.TrimPrefix(invoker.GetUrl().Path, "/"))
	for methodName, config := range restMethodConfig {
		// get method
		method := svc.Method()[methodName]
		types := method.ArgsType()
		ws := new(restful.WebService)
		ws.Path(config.Path).
			Produces(strings.Split(config.Produces, ",")...).
			Consumes(strings.Split(config.Consumes, ",")...).
			Route(ws.Method(config.MethodType).To(getFunc(methodName, invoker, types, config)))
		grs.container.Add(ws)
	}

}

func getFunc(methodName string, invoker protocol.Invoker, types []reflect.Type, config *rest_interface.RestMethodConfig) func(req *restful.Request, resp *restful.Response) {
	return func(req *restful.Request, resp *restful.Response) {
		var (
			err  error
			args []interface{}
		)
		args = getArgsFromRequest(req, types, config)
		result := invoker.Invoke(context.Background(), invocation.NewRPCInvocation(methodName, args, make(map[string]string, 0)))
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
func (grs *GoRestfulServer) UnDeploy(restMethodConfig map[string]*rest_interface.RestMethodConfig) {
	for _, config := range restMethodConfig {
		ws := new(restful.WebService)
		ws.Path(config.Path)
		err := grs.container.Remove(ws)
		if err != nil {
			logger.Warnf("[Go restful] Remove web service error:%v", err)
		}
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

func getArgsFromRequest(req *restful.Request, types []reflect.Type, config *rest_interface.RestMethodConfig) []interface{} {
	args := make([]interface{}, len(types))
	for i, t := range types {
		args[i] = reflect.Zero(t).Interface()
	}
	var (
		err   error
		param interface{}
		i64   int64
	)
	for k, v := range config.PathParamsMap {
		if k < 0 || k >= len(types) {
			logger.Errorf("[Go restful] Path param parse error, the args:%v doesn't exist", k)
			continue
		}
		t := types[k]
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
		} else if kind != reflect.String {
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
		if k < 0 || k > len(types) {
			logger.Errorf("[Go restful] Query param parse error, the args:%v doesn't exist", k)
			continue
		}
		t := types[k]
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

	if config.Body >= 0 && config.Body < len(types) {
		t := types[config.Body]
		kind := t.Kind()
		if kind == reflect.Ptr {
			t = t.Elem()
		}
		n := reflect.New(t)
		if n.CanInterface() {
			ni := n.Interface()
			if err := req.ReadEntity(ni); err != nil {
				logger.Errorf("[Go restful] Read body entity error:%v", err)
			} else {
				args[config.Body] = ni
			}
		}

	}
	for k, v := range config.HeadersMap {
		param := req.HeaderParameter(v)
		if k < 0 || k >= len(types) {
			logger.Errorf("[Go restful] Header param parse error, the args:%v doesn't exist", k)
			continue
		}
		t := types[k]
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

func GetNewGoRestfulServer() rest_interface.RestServer {
	return NewGoRestfulServer()
}
