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
	"context"
	"reflect"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

// MetadataServiceProxy actually is a  RPC stub which will only be used by client-side.
// If the metadata service is "local" metadata service in server side, which means that
// metadata service is RPC service too. So in client-side, if we want to get the metadata
// information, we must call metadata service .This is the stub, or proxy  for now, only
// GetMetadataInfo need to be implemented.
// TODO use ProxyFactory to create proxy
type MetadataServiceProxy struct {
	invkr protocol.Invoker
}

// nolint
func (m *MetadataServiceProxy) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]*common.URL, error) {
	siV := reflect.ValueOf(serviceInterface)
	gV := reflect.ValueOf(group)
	vV := reflect.ValueOf(version)
	pV := reflect.ValueOf(protocol)

	const methodName = "getExportedURLs"

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName(methodName),
		invocation.WithArguments([]interface{}{siV.Interface(), gV.Interface(), vV.Interface(), pV.Interface()}),
		invocation.WithReply(reflect.ValueOf(&[]interface{}{}).Interface()),
		invocation.WithAttachments(map[string]interface{}{constant.AsyncKey: "false"}),
		invocation.WithParameterValues([]reflect.Value{siV, gV, vV, pV}))

	res := m.invkr.Invoke(context.Background(), inv)
	if res.Error() != nil {
		logger.Errorf("could not get the metadata service from remote provider: %v", res.Error())
		return []*common.URL{}, nil
	}

	urlStrs := res.Result().([]string)
	ret := make([]*common.URL, 0, len(urlStrs))
	for _, v := range urlStrs {
		tempURL, err := common.NewURL(v)
		if err != nil {
			return []*common.URL{}, err
		}
		ret = append(ret, tempURL)
	}
	return ret, nil
}

// nolint
func (m *MetadataServiceProxy) GetExportedServiceURLs() ([]*common.URL, error) {
	logger.Error("you should never invoke this implementation")
	return nil, nil
}

// nolint
func (m *MetadataServiceProxy) GetMetadataServiceURL() (*common.URL, error) {
	logger.Error("you should never invoke this implementation")
	return nil, nil
}

// nolint
func (m *MetadataServiceProxy) SetMetadataServiceURL(*common.URL) error {
	logger.Error("you should never invoke this implementation")
	return nil
}

// nolint
func (m *MetadataServiceProxy) MethodMapper() map[string]string {
	return map[string]string{}
}

// nolint
func (m *MetadataServiceProxy) Reference() string {
	logger.Error("you should never invoke this implementation")
	return constant.MetadataServiceName
}

// nolint
func (m *MetadataServiceProxy) ServiceName() (string, error) {
	logger.Error("you should never invoke this implementation")
	return "", nil
}

// nolint
func (m *MetadataServiceProxy) ExportURL(url *common.URL) (bool, error) {
	logger.Error("you should never invoke this implementation")
	return false, nil
}

// nolint
func (m *MetadataServiceProxy) UnexportURL(url *common.URL) error {
	logger.Error("you should never invoke this implementation")
	return nil
}

// nolint
func (m *MetadataServiceProxy) SubscribeURL(url *common.URL) (bool, error) {
	logger.Error("you should never invoke this implementation")
	return false, nil
}

// nolint
func (m *MetadataServiceProxy) UnsubscribeURL(url *common.URL) error {
	logger.Error("you should never invoke this implementation")
	return nil
}

// nolint
func (m *MetadataServiceProxy) PublishServiceDefinition(url *common.URL) error {
	logger.Error("you should never invoke this implementation")
	return nil
}

// nolint
func (m *MetadataServiceProxy) GetSubscribedURLs() ([]*common.URL, error) {
	logger.Error("you should never invoke this implementation")
	return nil, nil
}

// nolint
func (m *MetadataServiceProxy) GetServiceDefinition(interfaceName string, group string, version string) (string, error) {
	logger.Error("you should never invoke this implementation")
	return "", nil
}

// nolint
func (m *MetadataServiceProxy) GetServiceDefinitionByServiceKey(serviceKey string) (string, error) {
	logger.Error("you should never invoke this implementation")
	return "", nil
}

// nolint
func (m *MetadataServiceProxy) RefreshMetadata(exportedRevision string, subscribedRevision string) (bool, error) {
	logger.Error("you should never invoke this implementation")
	return false, nil
}

// nolint
func (m *MetadataServiceProxy) Version() (string, error) {
	logger.Error("you should never invoke this implementation")
	return "", nil
}

// nolint
func (m *MetadataServiceProxy) GetMetadataInfo(revision string) (*common.MetadataInfo, error) {
	rV := reflect.ValueOf(revision)
	const methodName = "getMetadataInfo"
	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName(methodName),
		invocation.WithArguments([]interface{}{rV.Interface()}),
		invocation.WithReply(reflect.ValueOf(&common.MetadataInfo{}).Interface()),
		invocation.WithAttachments(map[string]interface{}{constant.AsyncKey: "false"}),
		invocation.WithParameterValues([]reflect.Value{rV}))
	res := m.invkr.Invoke(context.Background(), inv)
	if res.Error() != nil {
		logger.Errorf("could not get the metadata info from remote provider: %v", res.Error())
		return nil, res.Error()
	}
	metaDataInfo := res.Result().(*common.MetadataInfo)
	return metaDataInfo, nil
}
