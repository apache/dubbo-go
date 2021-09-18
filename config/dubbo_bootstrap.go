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

package config

import "net/http"

import (
	"dubbo.apache.org/dubbo-go/v3/common/logger"
)

func GetInstance(opts ...RootConfigOpt) *RootConfig {

	rc := &RootConfig{
		ConfigCenter:         &CenterConfig{},
		ServiceDiscoveries:   make(map[string]*ServiceDiscoveryConfig),
		MetadataReportConfig: &MetadataReportConfig{},
		Application:          GetApplicationInstance(),
		Registries:           make(map[string]*RegistryConfig),
		Protocols:            make(map[string]*ProtocolConfig),
		Provider:             NewProviderConfig(),
		Consumer:             NewConsumerConfig(),
		MetricConfig:         &MetricConfig{},
		Logger:               GetLoggerConfigInstance(),
	}
	for _, opt := range opts {
		opt(rc)
	}
	return rc
}

func (rc *RootConfig) Initialize() error {
	if err := rc.Logger.Init(); err != nil {
		return err
	}
	if err := initCenterConfig(rc); err != nil {
		logger.Infof("config center doesn't start. error is %s", err)
	}
	if err := rc.Application.Init(); err != nil {
		return err
	}
	if err := initProtocolsConfig(rc); err != nil {
		return err
	}
	if err := initRegistryConfig(rc); err != nil {
		return err
	}
	if err := initServiceDiscoveryConfig(rc); err != nil {
		return err
	}
	if err := rc.MetadataReportConfig.Init(rc); err != nil {
		return err
	}
	if err := initMetricConfig(rc); err != nil {
		return err
	}
	if err := initNetworkConfig(rc); err != nil {
		return err
	}
	if err := initRouterConfig(rc); err != nil {
		return err
	}
	// provider、consumer must last init
	if err := rc.Provider.Init(rc); err != nil {
		return err
	}
	if err := rc.Consumer.Init(rc); err != nil {
		return err
	}
	go func() {
		_ = http.ListenAndServe("0.0.0.0:6060", nil)
	}()
	return nil
}
