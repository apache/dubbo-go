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
	"context"
	"reflect"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

// actually it's RPC stub
// this will only be used by client-side
// if the metadata service is "local" metadata service in server side,
// which means that metadata service is RPC service too.
// so in client-side, if we want to get the metadata information,
// we must call metadata service
// this is the stub, or proxy
// for now, only GetExportedURLs need to be implemented
type MetadataServiceProxy struct {
	invkr        protocol.Invoker
	golangServer bool
}

func (m *MetadataServiceProxy) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]interface{}, error) {

	siV := reflect.ValueOf(serviceInterface)
	gV := reflect.ValueOf(group)
	vV := reflect.ValueOf(version)
	pV := reflect.ValueOf(protocol)

	const methodName = "getExportedURLs"

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName(methodName),
		invocation.WithArguments([]interface{}{siV.Interface(), gV.Interface(), vV.Interface(), pV.Interface()}),
		invocation.WithReply(reflect.ValueOf(&[]interface{}{}).Interface()),
		invocation.WithAttachments(map[string]interface{}{constant.ASYNC_KEY: "false"}),
		invocation.WithParameterValues([]reflect.Value{siV, gV, vV, pV}))

	res := m.invkr.Invoke(context.Background(), inv)
	if res.Error() != nil {
		logger.Errorf("could not get the metadata service from remote provider: %v", res.Error())
		return []interface{}{}, nil
	}

	urlStrs := res.Result().(*[]interface{})

	ret := make([]interface{}, 0, len(*urlStrs))

	for _, s := range *urlStrs {
		ret = append(ret, s)
	}
	return ret, nil
}

func (m *MetadataServiceProxy) MethodMapper() map[string]string {
	return map[string]string{}
}

func (m *MetadataServiceProxy) Reference() string {
	logger.Error("you should never invoke this implementation")
	return constant.METADATA_SERVICE_NAME
}

func (m *MetadataServiceProxy) ServiceName() (string, error) {
	logger.Error("you should never invoke this implementation")
	return "", nil
}

func (m *MetadataServiceProxy) ExportURL(url *common.URL) (bool, error) {
	logger.Error("you should never invoke this implementation")
	return false, nil
}

func (m *MetadataServiceProxy) UnexportURL(url *common.URL) error {
	logger.Error("you should never invoke this implementation")
	return nil
}

func (m *MetadataServiceProxy) SubscribeURL(url *common.URL) (bool, error) {
	logger.Error("you should never invoke this implementation")
	return false, nil
}

func (m *MetadataServiceProxy) UnsubscribeURL(url *common.URL) error {
	logger.Error("you should never invoke this implementation")
	return nil
}

func (m *MetadataServiceProxy) PublishServiceDefinition(url *common.URL) error {
	logger.Error("you should never invoke this implementation")
	return nil
}

func (m *MetadataServiceProxy) GetSubscribedURLs() ([]*common.URL, error) {
	logger.Error("you should never invoke this implementation")
	return nil, nil
}

func (m *MetadataServiceProxy) GetServiceDefinition(interfaceName string, group string, version string) (string, error) {
	logger.Error("you should never invoke this implementation")
	return "", nil
}

func (m *MetadataServiceProxy) GetServiceDefinitionByServiceKey(serviceKey string) (string, error) {
	logger.Error("you should never invoke this implementation")
	return "", nil
}

func (m *MetadataServiceProxy) RefreshMetadata(exportedRevision string, subscribedRevision string) (bool, error) {
	logger.Error("you should never invoke this implementation")
	return false, nil
}

func (m *MetadataServiceProxy) Version() (string, error) {
	logger.Error("you should never invoke this implementation")
	return "", nil
}
