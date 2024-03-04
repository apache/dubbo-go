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

// Package server provides APIs for registering services and starting an RPC server.
package server

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry/exposed_tmp"
)

// proServices are for internal services
var proServices = make([]*InternalService, 0, 16)
var proLock sync.Mutex

type Server struct {
	invoker protocol.Invoker
	info    *ServiceInfo

	cfg *ServerOptions

	svcOptsMap sync.Map
}

// ServiceInfo is meta info of a service
type ServiceInfo struct {
	InterfaceName string
	ServiceType   interface{}
	Methods       []MethodInfo
	Meta          map[string]interface{}
}

type infoInvoker struct {
	url       *common.URL
	base      *protocol.BaseInvoker
	info      *ServiceInfo
	svc       common.RPCService
	methodMap map[string]*MethodInfo
}

type ServiceDefinition struct {
	Handler interface{}
	Info    *ServiceInfo
	Opts    []ServiceOption
}

func (ii *infoInvoker) init() {
	url := ii.base.GetURL()
	if url.SubURL != nil {
		url = url.SubURL
	}
	ii.url = url
	methodMap := make(map[string]*MethodInfo)
	for i := range ii.info.Methods {
		methodMap[ii.info.Methods[i].Name] = &ii.info.Methods[i]
	}
	ii.methodMap = methodMap
}

func (ii *infoInvoker) GetURL() *common.URL {
	return ii.base.GetURL()
}

func (ii *infoInvoker) IsAvailable() bool {
	return ii.base.IsAvailable()
}

func (ii *infoInvoker) Destroy() {
	ii.base.Destroy()
}

func (ii *infoInvoker) Invoke(ctx context.Context, invocation protocol.Invocation) protocol.Result {
	name := invocation.MethodName()
	args := invocation.Arguments()
	result := new(protocol.RPCResult)
	if method, ok := ii.methodMap[name]; ok {
		res, err := method.MethodFunc(ctx, args, ii.svc)
		result.SetResult(res)
		result.SetError(err)
		return result
	}
	result.SetError(fmt.Errorf("no match method for %s", name))

	return result
}

func newInfoInvoker(url *common.URL, info *ServiceInfo, svc common.RPCService) protocol.Invoker {
	invoker := &infoInvoker{
		base: protocol.NewBaseInvoker(url),
		info: info,
		svc:  svc,
	}
	invoker.init()
	return invoker
}

// Register assemble invoker chains like ProviderConfig.Load, init a service per call
func (s *Server) Register(handler interface{}, info *ServiceInfo, opts ...ServiceOption) error {
	newSvcOpts, err := s.genSvcOpts(handler, opts...)
	if err != nil {
		return err
	}
	s.svcOptsMap.Store(newSvcOpts, info)
	return nil
}

func (s *Server) genSvcOpts(handler interface{}, opts ...ServiceOption) (*ServiceOptions, error) {
	if s.cfg == nil {
		return nil, errors.New("Server has not been initialized, please use NewServer() to create Server")
	}
	var svcOpts []ServiceOption
	appCfg := s.cfg.Application
	proCfg := s.cfg.Provider
	prosCfg := s.cfg.Protocols
	regsCfg := s.cfg.Registries
	// todo(DMwangnima): record the registered service
	newSvcOpts := defaultServiceOptions()
	if appCfg != nil {
		svcOpts = append(svcOpts,
			SetApplication(s.cfg.Application),
		)
	}
	if proCfg != nil {
		svcOpts = append(svcOpts,
			SetProvider(proCfg),
		)
	}
	if prosCfg != nil {
		svcOpts = append(svcOpts,
			SetProtocols(prosCfg),
		)
	}
	if regsCfg != nil {
		svcOpts = append(svcOpts,
			SetRegistries(regsCfg),
		)
	}
	// options passed by users have higher priority
	svcOpts = append(svcOpts, opts...)
	if err := newSvcOpts.init(s, svcOpts...); err != nil {
		return nil, err
	}
	newSvcOpts.Implement(handler)
	return newSvcOpts, nil
}

func (s *Server) exportServices() (err error) {
	s.svcOptsMap.Range(func(svcOptsRaw, infoRaw interface{}) bool {
		svcOpts := svcOptsRaw.(*ServiceOptions)
		if infoRaw == nil {
			err = svcOpts.ExportWithoutInfo()
		} else {
			info := infoRaw.(*ServiceInfo)
			err = svcOpts.ExportWithInfo(info)
		}
		if err != nil {
			logger.Errorf("export %s service failed, err: %s", svcOpts.Service.Interface, err)
			return false
		}
		return true
	})
	return err
}

func (s *Server) Serve() error {
	// the registryConfig in ServiceOptions and ServerOptions all need to init a metadataReporter,
	// when ServiceOptions.init() is called we don't know if a new registry config is set in the future use serviceOption
	if err := s.initRegistryMetadataReport(); err != nil {
		return err
	}
	opts := metadata.NewOptions(
		metadata.WithAppName(s.cfg.Application.Name),
		metadata.WithMetadataType(s.cfg.Application.MetadataType),
		metadata.WithPort(getMetadataPort(s.cfg)),
	)
	if err := opts.Init(); err != nil {
		return err
	}
	if err := s.exportServices(); err != nil {
		return err
	}
	if err := s.exportInternalServices(); err != nil {
		return err
	}
	if err := exposed_tmp.RegisterServiceInstance(); err != nil {
		return err
	}
	select {}
}

// In order to expose internal services
func (s *Server) exportInternalServices() error {
	cfg := &ServiceOptions{}
	cfg.Application = s.cfg.Application
	cfg.Provider = s.cfg.Provider
	cfg.Protocols = s.cfg.Protocols
	cfg.Registries = s.cfg.Registries

	services := make([]*InternalService, 0, len(proServices))

	proLock.Lock()
	defer proLock.Unlock()
	for _, service := range proServices {
		if service.Init == nil {
			return errors.New("[internal service]internal service init func is empty, please set the init func correctly")
		}
		sd, ok := service.Init(cfg)
		if !ok {
			logger.Infof("[internal service]%s service will not expose", service.Name)
			continue
		}
		newSvcOpts, err := s.genSvcOpts(sd.Handler, sd.Opts...)
		if err != nil {
			return err
		}
		service.svcOpts = newSvcOpts
		service.info = sd.Info
		services = append(services, service)
	}

	sort.Slice(services, func(i, j int) bool {
		return services[i].Priority < services[j].Priority
	})

	for _, service := range services {
		if service.BeforeExport != nil {
			service.BeforeExport(service.svcOpts)
		}
		err := service.svcOpts.ExportWithInfo(service.info)
		if service.AfterExport != nil {
			service.AfterExport(service.svcOpts, err)
		}
		if err != nil {
			logger.Errorf("[internal service]export %s service failed, err: %s", service.Name, err)
			return err
		}
	}

	return nil
}

// InternalService for dubbo internal services
type InternalService struct {
	// This is required
	// internal service name
	Name    string
	svcOpts *ServiceOptions
	info    *ServiceInfo
	// This is required
	// This options is service configuration
	// Return serviceDefinition and bool, where bool indicates whether it is exported
	Init func(options *ServiceOptions) (*ServiceDefinition, bool)
	// This options is InternalService.svcOpts itself
	BeforeExport func(options *ServiceOptions)
	// This options is InternalService.svcOpts itself
	AfterExport func(options *ServiceOptions, err error)
	// Priority of service exposure
	// Lower numbers have the higher priority
	// The default priority is 0
	// The metadata service is exposed at the end
	// If you have no requirements for the order of service exposure, you can use the default priority or not set
	Priority int
}

func getMetadataPort(opts *ServerOptions) int {
	port := opts.Application.MetadataServicePort
	if port == "" {
		protocolConfig, ok := opts.Protocols[constant.DefaultProtocol]
		if ok {
			port = protocolConfig.Port
		} else {
			logger.Warnf("[Metadata Service] Dubbo-go %s version's MetadataService only support dubbo protocol,"+
				"MetadataService will use random port",
				constant.Version)
		}
	}
	if port == "" {
		return 0
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		logger.Error("MetadataService port parse error %v, MetadataService will use random port", err)
		return 0
	}
	return p
}

func (s *Server) initRegistryMetadataReport() error {
	if len(s.cfg.Registries) > 0 {
		for id, reg := range s.cfg.Registries {
			if reg.UseAsMetaReport {
				opts, err := registryToReportOptions(id, reg)
				if err != nil {
					return err
				}
				if err := opts.Init(); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func registryToReportOptions(id string, rc *global.RegistryConfig) (*metadata.ReportOptions, error) {
	opts := metadata.NewReportOptions(
		metadata.WithRegistryId(id),
		metadata.WithProtocol(rc.Protocol),
		metadata.WithAddress(rc.Address),
		metadata.WithUsername(rc.Username),
		metadata.WithPassword(rc.Password),
		metadata.WithGroup(rc.Group),
		metadata.WithNamespace(rc.Namespace),
		metadata.WithParams(rc.Params),
	)
	if rc.Timeout != "" {
		timeout, err := time.ParseDuration(rc.Timeout)
		if err != nil {
			return nil, err
		}
		metadata.WithTimeout(timeout)(opts)
	}
	return opts, nil
}

type MethodInfo struct {
	Name           string
	Type           string
	ReqInitFunc    func() interface{}
	StreamInitFunc func(baseStream interface{}) interface{}
	MethodFunc     func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error)
	Meta           map[string]interface{}
}

func NewServer(opts ...ServerOption) (*Server, error) {
	newSrvOpts := defaultServerOptions()
	if err := newSrvOpts.init(opts...); err != nil {
		return nil, err
	}

	srv := &Server{
		cfg: newSrvOpts,
	}
	return srv, nil
}

func SetProServices(sd *InternalService) {
	if sd.Name == "" {
		logger.Warnf("[internal service]internal name is empty, please set internal name")
		return
	}
	proLock.Lock()
	defer proLock.Unlock()
	proServices = append(proServices, sd)
}
