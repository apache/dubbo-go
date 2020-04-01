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

func (grs *GoRestfulServer) Deploy(restMethodConfig *config.RestMethodConfig, routeFunc func(request server.RestServerRequest, response server.RestServerResponse)) {
	ws := new(restful.WebService)
	rf := func(req *restful.Request, resp *restful.Response) {
		routeFunc(NewGoRestfulRequestAdapter(req), resp)
	}
	ws.Path(restMethodConfig.Path).
		Produces(strings.Split(restMethodConfig.Produces, ",")...).
		Consumes(strings.Split(restMethodConfig.Consumes, ",")...).
		Route(ws.Method(restMethodConfig.MethodType).To(rf))
	grs.container.Add(ws)

}

func (grs *GoRestfulServer) UnDeploy(restMethodConfig *config.RestMethodConfig) {
	ws := new(restful.WebService)
	ws.Path(restMethodConfig.Path)
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

func GetNewGoRestfulServer() server.RestServer {
	return NewGoRestfulServer()
}

// Let user addFilter
// addFilter should before config.Load()
func AddGoRestfulServerFilter(filterFuc restful.FilterFunction) {
	filterSlice = append(filterSlice, filterFuc)
}

// Adapter about RestServerRequest
type GoRestfulRequestAdapter struct {
	server.RestServerRequest
	request *restful.Request
}

func NewGoRestfulRequestAdapter(request *restful.Request) *GoRestfulRequestAdapter {
	return &GoRestfulRequestAdapter{request: request}
}

func (grra *GoRestfulRequestAdapter) RawRequest() *http.Request {
	return grra.request.Request
}

func (grra *GoRestfulRequestAdapter) PathParameter(name string) string {
	return grra.request.PathParameter(name)
}

func (grra *GoRestfulRequestAdapter) PathParameters() map[string]string {
	return grra.request.PathParameters()
}

func (grra *GoRestfulRequestAdapter) QueryParameter(name string) string {
	return grra.request.QueryParameter(name)
}

func (grra *GoRestfulRequestAdapter) QueryParameters(name string) []string {
	return grra.request.QueryParameters(name)
}

func (grra *GoRestfulRequestAdapter) BodyParameter(name string) (string, error) {
	return grra.request.BodyParameter(name)
}

func (grra *GoRestfulRequestAdapter) HeaderParameter(name string) string {
	return grra.request.HeaderParameter(name)
}

func (grra *GoRestfulRequestAdapter) ReadEntity(entityPointer interface{}) error {
	return grra.request.ReadEntity(entityPointer)
}
