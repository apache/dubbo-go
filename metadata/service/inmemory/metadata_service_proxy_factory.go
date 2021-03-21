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

package inmemory

import (
	"encoding/json"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/metadata/service"
	"github.com/apache/dubbo-go/registry"
)

func init() {
	factory := service.NewBaseMetadataServiceProxyFactory(createProxy)
	extension.SetMetadataServiceProxyFactory(local, func() service.MetadataServiceProxyFactory {
		return factory
	})
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
	res := make([]*common.URL, 0, len(ps))
	sn := ins.GetServiceName()
	host := ins.GetHost()
	for protocol, params := range ps {

		convertedParams := make(map[string][]string, len(params))
		for k, v := range params {
			convertedParams[k] = []string{v}
		}

		u := common.NewURLWithOptions(common.WithIp(host),
			common.WithPath(constant.METADATA_SERVICE_NAME),
			common.WithProtocol(protocol),
			common.WithPort(params[constant.PORT_KEY]),
			common.WithParams(convertedParams),
			common.WithParamsValue(constant.GROUP_KEY, sn))
		res = append(res, u)
	}
	return res
}

// getMetadataServiceUrlParams this will convert the metadata service url parameters to map structure
// it looks like:
// {"dubbo":{"timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"20880"}}
func getMetadataServiceUrlParams(ins registry.ServiceInstance) map[string]map[string]string {
	ps := ins.GetMetadata()
	res := make(map[string]map[string]string, 2)
	if str, ok := ps[constant.METADATA_SERVICE_URL_PARAMS_PROPERTY_NAME]; ok && len(str) > 0 {

		err := json.Unmarshal([]byte(str), &res)
		if err != nil {
			logger.Errorf("could not parse the metadata service url parameters to map", err)
		}
	}
	return res
}
