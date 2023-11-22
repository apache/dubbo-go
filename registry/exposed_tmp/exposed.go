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

package exposed_tmp

import (
	"reflect"
	"strconv"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	metadataService "dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

// RegisterServiceInstance register service instance
func RegisterServiceInstance(applicationName string, tag string, metadataType string) {
	url := selectMetadataServiceExportedURL()
	if url == nil {
		return
	}
	instance, err := createInstance(url, applicationName, tag, metadataType)
	if err != nil {
		panic(err)
	}
	p := extension.GetProtocol(constant.RegistryProtocol)
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
		logger.Infof("Starting register instance address %v", instance)
		err := sdr.GetServiceDiscovery().Register(instance)
		if err != nil {
			panic(err)
		}
	}
}

// // nolint
func createInstance(url *common.URL, applicationName string, tag string, metadataType string) (registry.ServiceInstance, error) {
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
	metadata[constant.MetadataStorageTypePropertyName] = metadataType

	instance := &registry.DefaultServiceInstance{
		ServiceName: applicationName,
		Host:        host,
		Port:        int(port),
		ID:          host + constant.KeySeparator + url.Port,
		Enable:      true,
		Healthy:     true,
		Metadata:    metadata,
		Tag:         tag,
	}

	for _, cus := range extension.GetCustomizers() {
		cus.Customize(instance)
	}

	return instance, nil
}

// selectMetadataServiceExportedURL get already be exported url
func selectMetadataServiceExportedURL() *common.URL {
	var selectedUrl *common.URL
	urlList := metadataService.GlobalMetadataService.GetExportedServiceURLs()
	if len(urlList) == 0 {
		return nil
	}
	for _, url := range urlList {
		selectedUrl = url
		// rest first
		if url.Protocol == "rest" {
			break
		}
	}
	return selectedUrl
}
