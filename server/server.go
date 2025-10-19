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
	"sort"
	"strconv"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/dubboutil"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/registry/exposed_tmp"
)

// proServices are for internal services
var proServices = make([]*InternalService, 0, 16)
var proLock sync.Mutex

type Server struct {
	cfg *ServerOptions

	svcOptsMap sync.Map
}

// ServiceInfo Deprecated： common.ServiceInfo type alias, just for compatible with old generate pb.go file
type ServiceInfo = common.ServiceInfo

// MethodInfo Deprecated： common.MethodInfo type alias， just for compatible with old generate pb.go file
type MethodInfo = common.MethodInfo

type ServiceDefinition struct {
	Handler any
	Info    *common.ServiceInfo
	Opts    []ServiceOption
}

// Register assemble invoker chains like ProviderConfig.Load, init a service per call
func (s *Server) Register(handler any, info *common.ServiceInfo, opts ...ServiceOption) error {
	baseOpts := []ServiceOption{WithIDLMode(constant.IDL)}
	baseOpts = append(baseOpts, opts...)
	newSvcOpts, err := s.genSvcOpts(handler, baseOpts...)
	if err != nil {
		return err
	}
	s.svcOptsMap.Store(newSvcOpts, info)
	return nil
}

// RegisterService is for new Triple non-idl mode implement.
func (s *Server) RegisterService(handler any, opts ...ServiceOption) error {
	baseOpts := []ServiceOption{
		WithIDLMode(constant.NONIDL),
		WithInterface(common.GetReference(handler)),
	}
	baseOpts = append(baseOpts, opts...)
	newSvcOpts, err := s.genSvcOpts(handler, baseOpts...)
	if err != nil {
		return err
	}
	s.svcOptsMap.Store(newSvcOpts, nil)
	return nil
}

func (s *Server) genSvcOpts(handler any, opts ...ServiceOption) (*ServiceOptions, error) {
	if s.cfg == nil {
		return nil, errors.New("Server has not been initialized, please use NewServer() to create Server")
	}
	var svcOpts []ServiceOption
	appCfg := s.cfg.Application
	proCfg := s.cfg.Provider
	prosCfg := s.cfg.Protocols
	regsCfg := s.cfg.Registries
	// todo(DMwangnima): record the registered service
	// Record the registered service for debugging and monitoring
	interfaceName := common.GetReference(handler)
	logger.Infof("Registering service: %s", interfaceName)

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
	// Get service-level configuration items from provider.services configuration
	if proCfg != nil && proCfg.Services != nil {
		// Get the unique identifier of the handler (the default is the structure name or the alias set during registration)
		// Give priority to accurately finding the service configuration from the configuration based on the reference name (i.e. the handler registration name)
		svcCfg, ok := proCfg.Services[interfaceName]
		if !ok {
			//fallback: traverse matching interface fields
			for _, cfg := range proCfg.Services {
				if cfg.Interface == interfaceName {
					svcCfg = cfg
				}
			}
		}

		if svcCfg != nil {
			svcOpts = append(svcOpts,
				SetService(svcCfg),
			)
			logger.Infof("Injected options from provider.services for %s", interfaceName)
		} else {
			// Only warn if there are actually services configured but none match
			// This avoids unnecessary warnings when using new server API without config files
			if len(proCfg.Services) > 0 {
				logger.Warnf("No matching service config found for [%s]", interfaceName)
			}
		}
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
	s.svcOptsMap.Range(func(svcOptsRaw, infoRaw any) bool {
		svcOpts := svcOptsRaw.(*ServiceOptions)
		if info, ok := infoRaw.(*common.ServiceInfo); !ok || info == nil {
			err = svcOpts.ExportWithoutInfo()
		} else {
			// Add a method with a name of a different first-letter case
			// to achieve interoperability with java
			// TODO: The method name case sensitivity in Dubbo-java should be addressed.
			// We ought to make changes to handle this issue.
			var additionalMethods []common.MethodInfo
			for _, method := range info.Methods {
				newMethod := method
				newMethod.Name = dubboutil.SwapCaseFirstRune(method.Name)
				additionalMethods = append(additionalMethods, newMethod)
			}

			info.Methods = append(info.Methods, additionalMethods...)

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
	if err := metadata.InitRegistryMetadataReport(s.cfg.Registries); err != nil {
		return err
	}
	metadataOpts := metadata.NewOptions(
		metadata.WithAppName(s.cfg.Application.Name),
		metadata.WithMetadataType(s.cfg.Application.MetadataType),
		metadata.WithPort(getMetadataPort(s.cfg)),
		metadata.WithMetadataProtocol(s.cfg.Application.MetadataServiceProtocol),
	)
	if err := metadataOpts.Init(); err != nil {
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
	info    *common.ServiceInfo
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

func SetProviderServices(sd *InternalService) {
	if sd.Name == "" {
		logger.Warnf("[internal service]internal name is empty, please set internal name")
		return
	}
	proLock.Lock()
	defer proLock.Unlock()
	proServices = append(proServices, sd)
}
