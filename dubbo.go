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
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/global"
	"dubbo.apache.org/dubbo-go/v3/server"
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
	if conCfg != nil {
		// these options come from Consumer and Root.
		// for dubbo-go developers, referring config/ConsumerConfig.Init and config/ReferenceConfig
		cliOpts = append(cliOpts,
			client.WithReferenceConfig(
				global.WithReference_Filter(conCfg.Filter),
				global.WithReference_RegistryIDs(conCfg.RegistryIDs),
				global.WithReference_Protocol(conCfg.Protocol),
				global.WithReference_TracingKey(conCfg.TracingKey),
				global.WithReference_Check(conCfg.Check),
			),
			client.WithConsumerConfig(
				global.WithConsumer_MeshEnabled(conCfg.MeshEnabled),
				global.WithConsumer_AdaptiveService(conCfg.AdaptiveService),
				global.WithConsumer_ProxyFactory(conCfg.ProxyFactory),
			),
		)
	}
	if appCfg != nil {
		cliOpts = append(cliOpts, client.SetApplication(appCfg))
	}
	if regsCfg != nil {
		cliOpts = append(cliOpts, client.SetRegistries(regsCfg))
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
	trasCfg := ins.insOpts.Tracing
	if appCfg != nil {
		srvOpts = append(srvOpts,
			server.WithServer_ApplicationConfig(
				global.WithApplication_Name(appCfg.Name),
				global.WithApplication_Organization(appCfg.Organization),
				global.WithApplication_Module(appCfg.Module),
				global.WithApplication_Version(appCfg.Version),
				global.WithApplication_Owner(appCfg.Owner),
				global.WithApplication_Environment(appCfg.Environment),
			),
		)
	}
	if regsCfg != nil {
		srvOpts = append(srvOpts, server.SetServer_Registries(regsCfg))
	}
	if prosCfg != nil {
		srvOpts = append(srvOpts, server.SetServer_Protocols(prosCfg))
	}
	if trasCfg != nil {
		for key, tra := range trasCfg {
			srvOpts = append(srvOpts,
				server.WithServer_TracingConfig(key,
					global.WithTracing_Name(tra.Name),
					global.WithTracing_ServiceName(tra.ServiceName),
					global.WithTracing_Address(tra.Address),
					global.WithTracing_UseAgent(*tra.UseAgent),
				),
			)
		}
	}

	// options passed by users has higher priority
	srvOpts = append(srvOpts, opts...)

	srv, err := server.NewServer(srvOpts...)
	if err != nil {
		return nil, err
	}

	return srv, nil
}
