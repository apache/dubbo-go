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
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/global"
	"github.com/pkg/errors"
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
	// todo(DMwangnima): use slices to maintain options
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
		cliOpts = append(cliOpts,
			client.WithApplicationConfig(
				global.WithApplication_Name(appCfg.Name),
				global.WithApplication_Organization(appCfg.Organization),
				global.WithApplication_Module(appCfg.Module),
				global.WithApplication_Version(appCfg.Version),
				global.WithApplication_Owner(appCfg.Owner),
				global.WithApplication_Environment(appCfg.Environment),
				global.WithApplication_Group(appCfg.Group),
				global.WithApplication_MetadataType(appCfg.MetadataType),
			),
		)
	}
	if regsCfg != nil {
		for key, reg := range regsCfg {
			cliOpts = append(cliOpts,
				client.WithRegistryConfig(key,
					global.WithRegistry_Protocol(reg.Protocol),
					global.WithRegistry_Timeout(reg.Timeout),
					global.WithRegistry_Group(reg.Group),
					global.WithRegistry_Namespace(reg.Namespace),
					global.WithRegistry_TTL(reg.TTL),
					global.WithRegistry_Address(reg.Address),
					global.WithRegistry_Username(reg.Username),
					global.WithRegistry_Password(reg.Password),
					global.WithRegistry_Simplified(reg.Simplified),
					global.WithRegistry_Preferred(reg.Preferred),
					global.WithRegistry_Zone(reg.Zone),
					global.WithRegistry_Weight(reg.Weight),
					global.WithRegistry_Params(reg.Params),
					global.WithRegistry_RegistryType(reg.RegistryType),
					global.WithRegistry_UseAsMetaReport(reg.UseAsMetaReport),
					global.WithRegistry_UseAsConfigCenter(reg.UseAsConfigCenter),
				),
			)
		}
	}
	// options passed by users has higher priority
	cliOpts = append(cliOpts, opts...)

	cli, err := client.NewClient(cliOpts...)
	if err != nil {
		return nil, err
	}

	return cli, nil
}
