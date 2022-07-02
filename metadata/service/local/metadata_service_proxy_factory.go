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

package local

import (
	"encoding/json"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/service"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func init() {
	factory := service.NewBaseMetadataServiceProxyFactory(createProxy)
	extension.SetMetadataServiceProxyFactory(constant.DefaultKey, func() service.MetadataServiceProxyFactory {
		return factory
	})
}

var (
	factory service.MetadataServiceProxyFactory
	once    *sync.Once
)

func GetInMemoryMetadataServiceProxyFactory() service.MetadataServiceProxyFactory {
	once.Do(func() {
		factory = service.NewBaseMetadataServiceProxyFactory(createProxy)
	})
	return factory
}

// createProxy creates an instance of MetadataServiceProxy
// we read the metadata from ins.Metadata()
// and then create an Invoker instance
// also we will mark this proxy as golang's proxy
func createProxy(ins registry.ServiceInstance) service.MetadataService {
	urls := buildStandardMetadataServiceURL(ins)
	if len(urls) == 0 {
		logger.Errorf("metadata service urls not found, %v", ins)
		return nil
	}

	u := urls[0]
	p := extension.GetProtocol(u.Protocol)
	invoker := p.Refer(u)
	return &MetadataServiceProxy{
		invkr: invoker,
	}
}

// buildStandardMetadataServiceURL will use standard format to build the metadata service url.
func buildStandardMetadataServiceURL(ins registry.ServiceInstance) []*common.URL {
	ps := getMetadataServiceUrlParams(ins)
	if ps[constant.ProtocolKey] == "" {
		return nil
	}
	res := make([]*common.URL, 0, len(ps))
	sn := ins.GetServiceName()
	host := ins.GetHost()
	convertedParams := make(map[string][]string, len(ps))
	for k, v := range ps {
		convertedParams[k] = []string{v}
	}
	u := common.NewURLWithOptions(common.WithIp(host),
		common.WithPath(constant.MetadataServiceName),
		common.WithProtocol(ps[constant.ProtocolKey]),
		common.WithPort(ps[constant.PortKey]),
		common.WithParams(convertedParams),
		common.WithParamsValue(constant.GroupKey, sn),
		common.WithParamsValue(constant.InterfaceKey, constant.MetadataServiceName))
	res = append(res, u)

	return res
}

// getMetadataServiceUrlParams this will convert the metadata service url parameters to map structure
// it looks like:
// {"dubbo":{"timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"20880"}}
func getMetadataServiceUrlParams(ins registry.ServiceInstance) map[string]string {
	ps := ins.GetMetadata()
	res := make(map[string]string, 2)
	if str, ok := ps[constant.MetadataServiceURLParamsPropertyName]; ok && len(str) > 0 {

		err := json.Unmarshal([]byte(str), &res)
		if err != nil {
			logger.Errorf("could not parse the metadata service url parameters to map", err)
		}
	}
	return res
}
