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
	"github.com/pkg/errors"
)

// Instance is the highest layer conception that user could touch. It is mapped from RootConfig.
// When users want to inject global configurations and configure common modules for client layer
// and server layer, user-side code would be like this:
//
// ins, err := NewInstance()
// cli, err := ins.NewClient()
type Instance struct {
	rootCfg *RootConfig
}

// NewInstance receives RootOption and initializes RootConfig. There are some processing
// tasks during initialization.
func NewInstance(opts ...RootOption) (*Instance, error) {
	rootCfg := defaultRootConfig()
	if err := rootCfg.Init(opts...); err != nil {
		return nil, err
	}

	return &Instance{rootCfg: rootCfg}, nil
}

// NewClient is like client.NewClient, but inject configurations from RootConfig and
// ConsumerConfig
func (ins *Instance) NewClient(opts ...client.ReferenceOption) (*client.Client, error) {
	if ins == nil || ins.rootCfg == nil {
		return nil, errors.New("Instance has not been initialized")
	}

	var refOpts []client.ReferenceOption
	conCfg := ins.rootCfg.Consumer
	if conCfg != nil {
		// these options come from Consumer and Root.
		// for dubbo-go developers, referring config/ConsumerConfig.Init and config/ReferenceConfig
		refOpts = append(refOpts,
			client.WithFilter(conCfg.Filter),
			client.WithRegistryIDs(conCfg.RegistryIDs),
			client.WithProtocol(conCfg.Protocol),
			client.WithTracingKey(conCfg.TracingKey),
			client.WithCheck(conCfg.Check),
			client.WithMeshEnabled(conCfg.MeshEnabled),
			client.WithAdaptiveService(conCfg.AdaptiveService),
			client.WithProxyFactory(conCfg.ProxyFactory),
			client.WithApplication(ins.rootCfg.Application),
			client.WithRegistries(ins.rootCfg.Registries),
		)
	}
	refOpts = append(refOpts, opts...)

	cli, err := client.NewClient(refOpts...)
	if err != nil {
		return nil, err
	}

	return cli, nil
}
