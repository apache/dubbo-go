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
	"github.com/pkg/errors"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/server"
)

var (
	consumerServices = map[string]*client.ClientDefinition{}
	providerServices = map[string]*server.ServiceDefinition{}
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
	if conCfg != nil {
		if conCfg.Check {
			cliOpts = append(cliOpts, client.WithCheck())
		}
		// these options come from Consumer and Root.
		// for dubbo-go developers, referring config/ConsumerConfig.Init and config/ReferenceConfig
		cliOpts = append(cliOpts,
			client.WithFilter(conCfg.Filter),
			// todo(DMwangnima): deal with Protocol
			client.WithRegistryIDs(conCfg.RegistryIDs),
			// todo(DMwangnima): deal with TracingKey
			client.SetConsumer(conCfg),
		)
	}
	if appCfg != nil {
		cliOpts = append(cliOpts, client.SetApplication(appCfg))
	}
	if regsCfg != nil {
		cliOpts = append(cliOpts, client.SetRegistries(regsCfg))
	}
	if sdCfg != nil {
		cliOpts = append(cliOpts, client.SetShutdown(sdCfg))
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
	if appCfg != nil {
		srvOpts = append(srvOpts,
			server.SetServer_Application(appCfg),
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
		srvOpts = append(srvOpts, server.SetServer_Registries(regsCfg))
	}
	if prosCfg != nil {
		srvOpts = append(srvOpts, server.SetServer_Protocols(prosCfg))
	}
	if sdCfg != nil {
		srvOpts = append(srvOpts, server.SetServer_Shutdown(sdCfg))
	}

	// options passed by users have higher priority
	srvOpts = append(srvOpts, opts...)

	srv, err := server.NewServer(srvOpts...)

	if err != nil {
		return nil, err
	}
	return srv, nil
}

func (ins *Instance) Start() (err error) {
	startOnce.Do(func() {
		// todo gracefulShutdown
		if err = ins.LoadConsumer(); err != nil {
			return
		}
		if err = ins.LoadProvider(); err != nil {
			return
		}
	})
	return err
}

// LoadProvider loads the service provider.
func (ins *Instance) LoadProvider() error {
	srv, err := ins.NewServer()
	if err != nil {
		return err
	}
	// register services
	for _, definition := range providerServices {
		if err = srv.Register(definition.Handler, definition.Info, definition.Opts...); err != nil {
			return err
		}
	}
	return srv.Start()
}

// LoadConsumer loads the service consumer.
func (ins *Instance) LoadConsumer() error {
	cli, err := ins.NewClient()
	if err != nil {
		return err
	}
	// refer services
	for _, definition := range consumerServices {
		if err = cli.Init(definition.Info); err != nil {
			return err
		}
		definition.Info.ClientInjectFunc(definition.Svc, cli)
	}
	return err
}

// SetConsumerServiceWithInfo sets the consumer service with the client information.
func SetConsumerServiceWithInfo(svc common.RPCService, info *client.ClientInfo) {
	consumerServices[info.InterfaceName] = &client.ClientDefinition{
		Svc:  svc,
		Info: info,
	}
}

// SetProviderServiceWithInfo sets the provider service with the server information.
func SetProviderServiceWithInfo(svc common.RPCService, info *server.ServiceInfo) {
	providerServices[info.InterfaceName] = &server.ServiceDefinition{
		Handler: svc,
		Info:    info,
	}
}
