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

package dubbo

import (
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/server"
)

var (
	consumerServices = map[string]*client.ClientDefinition{}
	conLock          sync.RWMutex
	providerServices = map[string]*server.ServiceDefinition{}
	proLock          sync.RWMutex
	startOnce        sync.Once
)

// Instance is the highest layer conception that user could touch. It is mapped from RootConfig.
// When users want to inject global configurations and configure common modules for client layer
// and server layer, user-side code would be like this:
//
// ins, err := NewInstance()
// cli, err := ins.NewClient()
type Instance struct {
	insOpts *InstanceOptions
}

// NewInstance receives InstanceOption and initializes RootConfig. There are some processing
// tasks during initialization.
func NewInstance(opts ...InstanceOption) (*Instance, error) {
	newInsOpts := defaultInstanceOptions()
	if err := newInsOpts.init(opts...); err != nil {
		return nil, err
	}

	return &Instance{insOpts: newInsOpts}, nil
}

// NewClient is like client.NewClient, but inject configurations from RootConfig and
// ConsumerConfig
func (ins *Instance) NewClient(opts ...client.ClientOption) (*client.Client, error) {
	if ins == nil || ins.insOpts == nil {
		return nil, errors.New("Instance has not been initialized")
	}

	var cliOpts []client.ClientOption
	conCfg := ins.insOpts.Consumer
	appCfg := ins.insOpts.Application
	regsCfg := ins.insOpts.Registries
	sdCfg := ins.insOpts.Shutdown
	metricsCfg := ins.insOpts.Metrics
	otelCfg := ins.insOpts.Otel

	if conCfg != nil {
		if conCfg.Check {
			cliOpts = append(cliOpts, client.WithClientCheck())
		}
		// these options come from Consumer and Root.
		// for dubbo-go developers, referring config/ConsumerConfig.Init and config/ReferenceConfig
		cliOpts = append(cliOpts,
			client.WithClientFilter(conCfg.Filter),
			// todo(DMwangnima): deal with Protocol
			client.WithClientRegistryIDs(conCfg.RegistryIDs...),
			// todo(DMwangnima): deal with TracingKey
			client.SetClientConsumer(conCfg),
		)
	}
	if appCfg != nil {
		cliOpts = append(cliOpts, client.SetApplication(appCfg))
	}
	if regsCfg != nil {
		cliOpts = append(cliOpts, client.SetClientRegistries(regsCfg))
	}
	if sdCfg != nil {
		cliOpts = append(cliOpts, client.SetClientShutdown(sdCfg))
	}
	if metricsCfg != nil {
		cliOpts = append(cliOpts, client.SetClientMetrics(metricsCfg))
	}
	if otelCfg != nil {
		cliOpts = append(cliOpts, client.SetClientOtel(otelCfg))
	}

	// options passed by users has higher priority
	cliOpts = append(cliOpts, opts...)

	cli, err := client.NewClient(cliOpts...)
	if err != nil {
		return nil, err
	}

	return cli, nil
}

// NewServer is like server.NewServer, but inject configurations from RootConfig.
func (ins *Instance) NewServer(opts ...server.ServerOption) (*server.Server, error) {
	if ins == nil || ins.insOpts == nil {
		return nil, errors.New("Instance has not been initialized")
	}

	var srvOpts []server.ServerOption
	appCfg := ins.insOpts.Application
	regsCfg := ins.insOpts.Registries
	prosCfg := ins.insOpts.Protocols
	sdCfg := ins.insOpts.Shutdown
	metricsCfg := ins.insOpts.Metrics
	otelCfg := ins.insOpts.Otel

	if appCfg != nil {
		srvOpts = append(srvOpts,
			server.SetServerApplication(appCfg),
			//server.WithServer_ApplicationConfig(
			//	global.WithApplication_Name(appCfg.Name),
			//	global.WithApplication_Organization(appCfg.Organization),
			//	global.WithApplication_Module(appCfg.Module),
			//	global.WithApplication_Version(appCfg.Version),
			//	global.WithApplication_Owner(appCfg.Owner),
			//	global.WithApplication_Environment(appCfg.Environment),
			//),
		)
	}
	if regsCfg != nil {
		srvOpts = append(srvOpts, server.SetServerRegistries(regsCfg))
	}
	if prosCfg != nil {
		srvOpts = append(srvOpts, server.SetServerProtocols(prosCfg))
	}
	if sdCfg != nil {
		srvOpts = append(srvOpts, server.SetServerShutdown(sdCfg))
	}
	if metricsCfg != nil {
		srvOpts = append(srvOpts, server.SetServerMetrics(metricsCfg))
	}
	if otelCfg != nil {
		srvOpts = append(srvOpts, server.SetServerOtel(otelCfg))
	}

	// options passed by users have higher priority
	srvOpts = append(srvOpts, opts...)

	srv, err := server.NewServer(srvOpts...)

	if err != nil {
		return nil, err
	}
	return srv, nil
}

func (ins *Instance) start() (err error) {
	startOnce.Do(func() {
		if err = ins.loadConsumer(); err != nil {
			return
		}
		if err = ins.loadProvider(); err != nil {
			return
		}
	})
	return err
}

// loadProvider loads the service provider.
func (ins *Instance) loadProvider() error {
	var srvOpts []server.ServerOption
	if ins.insOpts.Provider != nil {
		srvOpts = append(srvOpts, server.SetServerProvider(ins.insOpts.Provider))
	}
	srv, err := ins.NewServer(srvOpts...)
	if err != nil {
		return err
	}
	// register services
	proLock.RLock()
	defer proLock.RUnlock()
	for _, definition := range providerServices {
		if err = srv.Register(definition.Handler, definition.Info, definition.Opts...); err != nil {
			return err
		}
	}
	go func() {
		if err = srv.Serve(); err != nil {
			logger.Fatalf("Failed to start server, err: %v", err)
		}
	}()
	return nil
}

// loadConsumer loads the service consumer.
func (ins *Instance) loadConsumer() error {
	cli, err := ins.NewClient()
	if err != nil {
		return err
	}
	// refer services
	conLock.RLock()
	defer conLock.RUnlock()
	for intfName, definition := range consumerServices {
		conn, dialErr := cli.DialWithInfo(intfName, definition.Info)
		if dialErr != nil {
			return dialErr
		}
		definition.Info.ConnectionInjectFunc(definition.Svc, conn)
	}
	return nil
}

// SetConsumerServiceWithInfo sets the consumer service with the client information.
func SetConsumerServiceWithInfo(svc common.RPCService, info *client.ClientInfo) {
	conLock.Lock()
	defer conLock.Unlock()
	consumerServices[info.InterfaceName] = &client.ClientDefinition{
		Svc:  svc,
		Info: info,
	}
}

// SetProviderServiceWithInfo sets the provider service with the server information.
func SetProviderServiceWithInfo(svc common.RPCService, info *server.ServiceInfo) {
	proLock.Lock()
	defer proLock.Unlock()
	providerServices[info.InterfaceName] = &server.ServiceDefinition{
		Handler: svc,
		Info:    info,
	}
}
