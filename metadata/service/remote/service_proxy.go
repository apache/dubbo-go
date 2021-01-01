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

package remote

import (
	"strings"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config/instance"
	"github.com/apache/dubbo-go/metadata/identifier"
	"github.com/apache/dubbo-go/metadata/report"
	"github.com/apache/dubbo-go/metadata/service"
	"github.com/apache/dubbo-go/registry"
)

type metadataServiceProxy struct {
	serviceName string
	revision    string
	report      report.MetadataReport
}

func (m *metadataServiceProxy) Reference() string {
	return constant.METADATA_SERVICE_NAME
}

func (m *metadataServiceProxy) ServiceName() (string, error) {
	return m.serviceName, nil
}

func (m *metadataServiceProxy) ExportURL(url *common.URL) (bool, error) {
	logger.Error("you should never invoke this implementation")
	return true, nil
}

func (m *metadataServiceProxy) UnexportURL(url *common.URL) error {
	logger.Error("you should never invoke this implementation")
	return nil
}

func (m *metadataServiceProxy) SubscribeURL(url *common.URL) (bool, error) {
	logger.Error("you should never invoke this implementation")
	return true, nil
}

func (m *metadataServiceProxy) UnsubscribeURL(url *common.URL) error {
	logger.Error("you should never invoke this implementation")
	return nil
}

func (m *metadataServiceProxy) PublishServiceDefinition(url *common.URL) error {
	logger.Error("you should never invoke this implementation")
	return nil
}

func (m *metadataServiceProxy) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]interface{}, error) {
	urls, err := m.report.GetExportedURLs(&identifier.ServiceMetadataIdentifier{
		BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
			ServiceInterface: serviceInterface,
			Version:          version,
			Group:            group,
			Side:             constant.PROVIDER_PROTOCOL,
		},
		Revision: m.revision,
		Protocol: protocol,
	})

	if err != nil {
		return []interface{}{}, nil
	}
	var res []*common.URL
	for _, s := range urls {
		u, err := common.NewURL(s)
		if err != nil {
			logger.Errorf("could not parse the url string to URL structure", err)
			continue
		}
		res = append(res, u)
	}
	return service.ConvertURLArrToIntfArr(res), nil
}

func (m *metadataServiceProxy) MethodMapper() map[string]string {
	return map[string]string{}
}

func (m *metadataServiceProxy) GetSubscribedURLs() ([]*common.URL, error) {
	logger.Error("you should never invoke this implementation")
	return nil, nil
}

func (m *metadataServiceProxy) GetServiceDefinition(interfaceName string, group string, version string) (string, error) {
	return m.report.GetServiceDefinition(&identifier.MetadataIdentifier{
		BaseMetadataIdentifier: identifier.BaseMetadataIdentifier{
			ServiceInterface: interfaceName,
			Group:            group,
			Version:          version,
			Side:             constant.PROVIDER_PROTOCOL,
		},
		Application: m.serviceName,
	})
}

func (m *metadataServiceProxy) GetServiceDefinitionByServiceKey(serviceKey string) (string, error) {
	params := parse(serviceKey)
	return m.GetServiceDefinition(params[0], params[1], params[2])
}

func (m *metadataServiceProxy) RefreshMetadata(exportedRevision string, subscribedRevision string) (bool, error) {
	logger.Error("you should never invoke this implementation")
	return true, nil
}

func (m metadataServiceProxy) Version() (string, error) {
	return version, nil
}

func newMetadataServiceProxy(ins registry.ServiceInstance) service.MetadataService {
	revision := ins.GetMetadata()[constant.EXPORTED_SERVICES_REVISION_PROPERTY_NAME]
	if len(revision) == 0 {
		revision = constant.DEFAULT_REVIESION
	}

	return &metadataServiceProxy{
		serviceName: ins.GetServiceName(),
		revision:    revision,
		report:      instance.GetMetadataReportInstance(),
	}
}

func parse(key string) []string {
	arr := make([]string, 3, 3)
	tmp := strings.SplitN(key, "/", 2)
	if len(tmp) > 1 {
		arr[0] = tmp[0]
		key = tmp[1]
	}
	tmp = strings.SplitN(key, "/", 2)
	if len(tmp) > 1 {
		arr[2] = tmp[1]
		key = tmp[0]
	}
	arr[1] = key
	return arr
}
