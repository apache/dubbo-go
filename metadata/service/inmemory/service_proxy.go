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

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
)

type MetadataServiceProxy struct {
	invkr protocol.Invoker
}

func (m *MetadataServiceProxy) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]common.URL, error) {

	siV := reflect.ValueOf(serviceInterface)
	gV := reflect.ValueOf(group)
	vV := reflect.ValueOf(version)
	pV := reflect.ValueOf(protocol)

	inv := invocation.NewRPCInvocationWithOptions(invocation.WithMethodName("getExportedURLs"),
		invocation.WithArguments([]interface{}{siV.Interface(), gV.Interface(), vV.Interface(), pV.Interface()}),
		invocation.WithReply(reflect.ValueOf(&[]interface{}{}).Interface()),
		invocation.WithAttachments(map[string]string{constant.ASYNC_KEY: "false"}),
		invocation.WithParameterValues([]reflect.Value{siV, gV, vV, pV}))

	res := m.invkr.Invoke(context.Background(), inv)
	if res.Error() != nil {
		logger.Errorf("could not get the metadata service from remote provider: %v", res.Error())
	}

	urlStrs := res.Result().(*[]interface{})

	ret := make([]common.URL, 0, len(*urlStrs))

	for _, s := range *urlStrs {
		u, err := common.NewURL(s.(string))
		if err != nil {
			logger.Errorf("could not convert the string to URL: %s", s)
			continue
		}
		ret = append(ret, u)
	}
	return ret, nil
}

func (m *MetadataServiceProxy) Reference() string {
	panic("implement me")
}

func (m *MetadataServiceProxy) ServiceName() (string, error) {
	panic("implement me")
}

func (m *MetadataServiceProxy) ExportURL(url common.URL) (bool, error) {
	panic("implement me")
}

func (m *MetadataServiceProxy) UnexportURL(url common.URL) error {
	panic("implement me")
}

func (m *MetadataServiceProxy) SubscribeURL(url common.URL) (bool, error) {
	panic("implement me")
}

func (m *MetadataServiceProxy) UnsubscribeURL(url common.URL) error {
	panic("implement me")
}

func (m *MetadataServiceProxy) PublishServiceDefinition(url common.URL) error {
	panic("implement me")
}

func (m *MetadataServiceProxy) GetSubscribedURLs() ([]common.URL, error) {
	panic("implement me")
}

func (m *MetadataServiceProxy) GetServiceDefinition(interfaceName string, group string, version string) (string, error) {
	panic("implement me")
}

func (m *MetadataServiceProxy) GetServiceDefinitionByServiceKey(serviceKey string) (string, error) {
	panic("implement me")
}

func (m *MetadataServiceProxy) RefreshMetadata(exportedRevision string, subscribedRevision string) (bool, error) {
	panic("implement me")
}

func (m *MetadataServiceProxy) Version() (string, error) {
	panic("implement me")
}

type MetadataServiceStub struct {
	GetExportedURLs func(serviceInterface string, group string, version string, protocol string) ([]interface{}, error)
}

func (m *MetadataServiceStub) Reference() string {
	return constant.METADATA_SERVICE_NAME
}
