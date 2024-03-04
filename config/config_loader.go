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

import (
	"errors"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"github.com/knadh/koanf"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/logger/zap"
)

var (
	rootConfig = NewRootConfigBuilder().Build()
)

func init() {
	log := zap.NewDefault()
	logger.SetLogger(log)
}

func Load(opts ...LoaderConfOption) error {
	// conf
	conf := NewLoaderConf(opts...)
	if conf.rc == nil {
		koan := GetConfigResolver(conf)
		koan = conf.MergeConfig(koan)
		if err := koan.UnmarshalWithConf(rootConfig.Prefix(),
			rootConfig, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
			return err
		}
	} else {
		rootConfig = conf.rc
	}

	if err := rootConfig.Init(); err != nil {
		return err
	}
	return nil
}

func check() error {
	if rootConfig == nil {
		return errors.New("execute the config.Load() method first")
	}
	return nil
}

// GetRPCService get rpc service for consumer
func GetRPCService(name string) common.RPCService {
	return rootConfig.Consumer.References[name].GetRPCService()
}

// RPCService create rpc service for consumer
func RPCService(service common.RPCService) {
	ref := common.GetReference(service)
	rootConfig.Consumer.References[ref].Implement(service)
}

// GetMetricConfig find the MetricsConfig
// if it is nil, create a new one
// we use double-check to reduce race condition
// In general, it will be locked 0 or 1 time.
// So you don't need to worry about the race condition
func GetMetricConfig() *MetricsConfig {
	// todo
	//if GetBaseConfig().Metrics == nil {
	//	configAccessMutex.Lock()
	//	defer configAccessMutex.Unlock()
	//	if GetBaseConfig().Metrics == nil {
	//		GetBaseConfig().Metrics = &metric.Metrics{}
	//	}
	//}
	//return GetBaseConfig().Metrics
	return rootConfig.Metrics
}

func GetTracingConfig(tracingKey string) *TracingConfig {
	return rootConfig.Tracing[tracingKey]
}

func GetMetadataReportConfg() *MetadataReportConfig {
	return rootConfig.MetadataReport
}

func IsProvider() bool {
	return len(rootConfig.Provider.Services) > 0
}
