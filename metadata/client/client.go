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

package client

import (
	"context"
	"encoding/json"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	metadataInstance "dubbo.apache.org/dubbo-go/v3/metadata/report/instance"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

func GetMetadataFromMetadataReport(revision string, instance registry.ServiceInstance) (*info.MetadataInfo, error) {
	report := metadataInstance.GetMetadataReport()
	return report.GetAppMetadata(instance.GetServiceName(), revision)
}

func GetMetadataFromRpc(revision string, instance registry.ServiceInstance) (*info.MetadataInfo, error) {
	service, destroy := createRpcClient(instance)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5000))
	defer cancel()
	defer destroy()
	return service.GetMetadataInfo(ctx, revision)
}

type metadataService struct {
	GetExportedURLs       func(context context.Context, serviceInterface string, group string, version string, protocol string) ([]*common.URL, error) `dubbo:"getExportedURLs"`
	GetMetadataInfo       func(context context.Context, revision string) (*info.MetadataInfo, error)                                                   `dubbo:"getMetadataInfo"`
	GetMetadataServiceURL func(context context.Context) (*common.URL, error)
	GetSubscribedURLs     func(context context.Context) ([]*common.URL, error)
	Version               func(context context.Context) (string, error)
}

func createRpcClient(instance registry.ServiceInstance) (*metadataService, func()) {
	params := getMetadataServiceUrlParams(instance.GetMetadata()[constant.MetadataServiceURLParamsPropertyName])
	url := buildMetadataServiceURL(instance.GetServiceName(), instance.GetHost(), params)
	return createRpcClientByUrl(url)
}

func createRpcClientByUrl(url *common.URL) (*metadataService, func()) {
	rpcService := &metadataService{}
	invoker := extension.GetProtocol(constant.Dubbo).Refer(url)
	proxy := extension.GetProxyFactory("").GetProxy(invoker, url)
	proxy.Implement(rpcService)
	destroy := func() {
		invoker.Destroy()
	}
	return rpcService, destroy
}

// buildMetadataServiceURL will use standard format to build the metadata service url.
func buildMetadataServiceURL(serviceName string, host string, ps map[string]string) *common.URL {
	if ps[constant.ProtocolKey] == "" {
		return nil
	}
	convertedParams := make(map[string][]string, len(ps))
	for k, v := range ps {
		convertedParams[k] = []string{v}
	}
	u := common.NewURLWithOptions(common.WithIp(host),
		common.WithPath(constant.MetadataServiceName),
		common.WithProtocol(ps[constant.ProtocolKey]),
		common.WithPort(ps[constant.PortKey]),
		common.WithParams(convertedParams),
		common.WithParamsValue(constant.GroupKey, serviceName),
		common.WithParamsValue(constant.InterfaceKey, constant.MetadataServiceName))

	return u
}

// getMetadataServiceUrlParams this will convert the metadata service url parameters to map structure
// it looks like:
// {"dubbo":{"timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"20880"}}
func getMetadataServiceUrlParams(jsonStr string) map[string]string {
	res := make(map[string]string, 2)
	if len(jsonStr) > 0 {
		err := json.Unmarshal([]byte(jsonStr), &res)
		if err != nil {
			logger.Errorf("could not parse the metadata service url parameters to map", err)
		}
	}
	return res
}
