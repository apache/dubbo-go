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
	"fmt"
	"reflect"
	"strconv"
)

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/json"
	"github.com/knadh/koanf/parsers/toml"
	yaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

var (
	rootConfig = GetInstance()
	maxWait    = 3
)

func Load(opts ...LoaderConfOption) error {
	// conf
	conf := NewLoaderConf(opts...)
	koan := getKoanf(conf)
	if err := koan.UnmarshalWithConf(rootConfig.Prefix(),
		rootConfig, koanf.UnmarshalConf{Tag: "yaml"}); err != nil {
		return err
	}
	if err := rootConfig.Init(); err != nil {
		return err
	}
	rootConfig.Start()
	return nil
}

func check() error {
	if rootConfig == nil {
		return errors.New("execute the config.Load() method first")
	}
	return nil
}

func getKoanf(conf *loaderConf) *koanf.Koanf {
	var (
		k   *koanf.Koanf
		err error
	)
	k = koanf.New(conf.delim)

	switch conf.genre {
	case "yaml", "yml":
		err = k.Load(file.Provider(conf.path), yaml.Parser())
	case "json":
		err = k.Load(file.Provider(conf.path), json.Parser())
	case "toml":
		err = k.Load(file.Provider(conf.path), toml.Parser())
	default:
		err = errors.New(fmt.Sprintf("Unsupported %s file type", conf.genre))
	}

	if err != nil {
		panic(err)
	}
	return k
}

// registerServiceInstance register service instance
func registerServiceInstance() {
	url := selectMetadataServiceExportedURL()
	if url == nil {
		return
	}
	instance, err := createInstance(url)
	if err != nil {
		panic(err)
	}
	p := extension.GetProtocol(constant.REGISTRY_KEY)
	var rp registry.RegistryFactory
	var ok bool
	if rp, ok = p.(registry.RegistryFactory); !ok {
		panic("dubbo registry protocol{" + reflect.TypeOf(p).String() + "} is invalid")
	}
	rs := rp.GetRegistries()
	for _, r := range rs {
		var sdr registry.ServiceDiscoveryHolder
		if sdr, ok = r.(registry.ServiceDiscoveryHolder); !ok {
			continue
		}
		// publish app level data to registry
		err := sdr.GetServiceDiscovery().Register(instance)
		if err != nil {
			panic(err)
		}
	}
	// publish metadata to remote
	if GetApplicationConfig().MetadataType == constant.REMOTE_METADATA_STORAGE_TYPE {
		if remoteMetadataService, err := extension.GetRemoteMetadataService(); err == nil {
			remoteMetadataService.PublishMetadata(GetApplicationConfig().Name)
		}
	}

	if GetApplicationConfig().MetadataType == constant.REMOTE_METADATA_STORAGE_TYPE {
		// publish metadata to remote
		if remoteMetadataService, err := extension.GetRemoteMetadataService(); err == nil {
			remoteMetadataService.PublishMetadata(GetApplicationConfig().Name)
		}
	}
}

//
//// nolint
func createInstance(url *common.URL) (registry.ServiceInstance, error) {
	appConfig := GetApplicationConfig()
	port, err := strconv.ParseInt(url.Port, 10, 32)
	if err != nil {
		return nil, perrors.WithMessage(err, "invalid port: "+url.Port)
	}

	host := url.Ip
	if len(host) == 0 {
		host = common.GetLocalIp()
	}

	// usually we will add more metadata
	metadata := make(map[string]string, 8)
	metadata[constant.METADATA_STORAGE_TYPE_PROPERTY_NAME] = appConfig.MetadataType

	instance := &registry.DefaultServiceInstance{
		ServiceName: appConfig.Name,
		Host:        host,
		Port:        int(port),
		ID:          host + constant.KEY_SEPARATOR + url.Port,
		Enable:      true,
		Healthy:     true,
		Metadata:    metadata,
	}

	for _, cus := range extension.GetCustomizers() {
		cus.Customize(instance)
	}

	return instance, nil
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

// GetMetricConfig find the MetricConfig
// if it is nil, create a new one
// we use double-check to reduce race condition
// In general, it will be locked 0 or 1 time.
// So you don't need to worry about the race condition
func GetMetricConfig() *MetricConfig {
	// todo
	//if GetBaseConfig().MetricConfig == nil {
	//	configAccessMutex.Lock()
	//	defer configAccessMutex.Unlock()
	//	if GetBaseConfig().MetricConfig == nil {
	//		GetBaseConfig().MetricConfig = &metric.MetricConfig{}
	//	}
	//}
	//return GetBaseConfig().MetricConfig
	return rootConfig.MetricConfig
}

func GetMetadataReportConfg() *MetadataReportConfig {
	return rootConfig.MetadataReportConfig
}

func IsProvider() bool {
	// FixME
	return rootConfig.Provider != nil
}
