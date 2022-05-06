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
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/config"
)

var filterSlice []restful.FilterFunction

// GoRestfulServer a rest server implement by go-restful
type GoRestfulServer struct {
	srv *http.Server
	ws  *restful.WebService
}

// NewGoRestfulServer a constructor of GoRestfulServer
func NewGoRestfulServer() RestServer {
	return &GoRestfulServer{}
}

// Start go-restful server
// It will add all go-restful filters
func (grs *GoRestfulServer) Start(url *common.URL) {
	container := restful.NewContainer()
	for _, filter := range filterSlice {
		container.Filter(filter)
	}
	grs.srv = &http.Server{
		Handler: container,
	}
	grs.ws = &restful.WebService{}
	grs.ws.Path("/")
	grs.ws.SetDynamicRoutes(true)
	container.Add(grs.ws)
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

// Deploy is to Publish a http api in go-restful server
// The routeFunc should be invoked when the server receive a request
func (grs *GoRestfulServer) Deploy(restMethodConfig *config.RestMethodConfig, routeFunc func(request RestServerRequest, response RestServerResponse)) {
	rf := func(req *restful.Request, resp *restful.Response) {
		routeFunc(NewGoRestfulRequestAdapter(req), resp)
	}
	grs.ws.Route(grs.ws.Method(restMethodConfig.MethodType).
		Produces(strings.Split(restMethodConfig.Produces, ",")...).
		Consumes(strings.Split(restMethodConfig.Consumes, ",")...).
		Path(restMethodConfig.Path).To(rf))
}

// UnDeploy is to Delete a http api in go-restful server
func (grs *GoRestfulServer) UnDeploy(restMethodConfig *config.RestMethodConfig) {
	err := grs.ws.RemoveRoute(restMethodConfig.Path, restMethodConfig.MethodType)
	if err != nil {
		logger.Warnf("[Go restful] Remove web service error:%v", err)
	}
}

// Destroy the go-restful server
func (grs *GoRestfulServer) Destroy() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := grs.srv.Shutdown(ctx); err != nil {
		logger.Errorf("[Go Restful] Server Shutdown:", err)
	}
	logger.Infof("[Go Restful] Server exiting")
}

// AddGoRestfulServerFilter let user add the http server of go-restful
// addFilter should before config.Load()
func AddGoRestfulServerFilter(filterFuc restful.FilterFunction) {
	filterSlice = append(filterSlice, filterFuc)
}

// GoRestfulRequestAdapter a adapter struct about RestServerRequest
type GoRestfulRequestAdapter struct {
	RestServerRequest
	request *restful.Request
}

// NewGoRestfulRequestAdapter a constructor of GoRestfulRequestAdapter
func NewGoRestfulRequestAdapter(request *restful.Request) *GoRestfulRequestAdapter {
	return &GoRestfulRequestAdapter{request: request}
}

// RawRequest a adapter function of server.RestServerRequest's RawRequest
func (grra *GoRestfulRequestAdapter) RawRequest() *http.Request {
	return grra.request.Request
}

// PathParameter a adapter function of server.RestServerRequest's PathParameter
func (grra *GoRestfulRequestAdapter) PathParameter(name string) string {
	return grra.request.PathParameter(name)
}

// PathParameters a adapter function of server.RestServerRequest's QueryParameter
func (grra *GoRestfulRequestAdapter) PathParameters() map[string]string {
	return grra.request.PathParameters()
}

// QueryParameter a adapter function of server.RestServerRequest's QueryParameters
func (grra *GoRestfulRequestAdapter) QueryParameter(name string) string {
	return grra.request.QueryParameter(name)
}

// QueryParameters a adapter function of server.RestServerRequest's QueryParameters
func (grra *GoRestfulRequestAdapter) QueryParameters(name string) []string {
	return grra.request.QueryParameters(name)
}

// BodyParameter a adapter function of server.RestServerRequest's BodyParameter
func (grra *GoRestfulRequestAdapter) BodyParameter(name string) (string, error) {
	return grra.request.BodyParameter(name)
}

// HeaderParameter a adapter function of server.RestServerRequest's HeaderParameter
func (grra *GoRestfulRequestAdapter) HeaderParameter(name string) string {
	return grra.request.HeaderParameter(name)
}

// ReadEntity a adapter func of server.RestServerRequest's ReadEntity
func (grra *GoRestfulRequestAdapter) ReadEntity(entityPointer interface{}) error {
	return grra.request.ReadEntity(entityPointer)
}
