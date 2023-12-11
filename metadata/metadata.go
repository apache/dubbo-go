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

// Package metadata collects and exposes information of all services for service discovery purpose.
package metadata

import (
	"fmt"
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
	gxnet "github.com/dubbogo/gost/net"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

var (
	metadataService MetadataService = &DefaultMetadataService{}
	exportOnce      sync.Once
)

func ExportMetadataService(app, metadataType string) {
	exportOnce.Do(func() {
		if metadataType != constant.RemoteMetadataStorageType {
			exporter := &ServiceExporter{app: app, metadataType: metadataType, service: metadataService}
			err := exporter.Export()
			if err != nil {
				logger.Errorf("export metadata service failed, got error %#v", err)
			}
		}
	})
}

func GetMetadataService() MetadataService {
	return metadataService
}

// ServiceExporter is the ConfigurableMetadataServiceExporter which implement MetadataServiceExporter interface
type ServiceExporter struct {
	app, metadataType string
	service           MetadataService
	protocolExporter  protocol.Exporter
}

// Export will export the metadataService
func (exporter *ServiceExporter) Export() error {
	version, _ := exporter.service.Version()
	tcp, err := gxnet.ListenOnTCPRandomPort("")
	if err != nil {
		panic(perrors.New(fmt.Sprintf("Get tcp port error, err is {%v}", err)))
	}
	err = tcp.Close()
	if err != nil {
		panic(perrors.New(fmt.Sprintf("Close tcp port error, err is {%v}", err)))
	}
	ivkURL := common.NewURLWithOptions(
		common.WithPath(constant.MetadataServiceName),
		common.WithProtocol(constant.DefaultProtocol),
		common.WithPort(strings.Split(tcp.Addr().String(), ":")[1]),
		common.WithParamsValue(constant.GroupKey, exporter.app),
		common.WithParamsValue(constant.SerializationKey, constant.Hessian2Serialization),
		common.WithParamsValue(constant.ReleaseKey, constant.Version),
		common.WithParamsValue(constant.VersionKey, version),
		common.WithParamsValue(constant.InterfaceKey, constant.MetadataServiceName),
		common.WithParamsValue(constant.BeanNameKey, constant.SimpleMetadataServiceName),
		common.WithParamsValue(constant.MetadataTypeKey, exporter.metadataType),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)
	methods, err := common.ServiceMap.Register(ivkURL.Interface(), ivkURL.Protocol, ivkURL.Group(), ivkURL.Version(), exporter.service)
	if err != nil {
		formatErr := perrors.Errorf("The service %v needExport the protocol %v error! Error message is %v.",
			ivkURL.Interface(), ivkURL.Protocol, err.Error())
		logger.Errorf(formatErr.Error())
		return formatErr
	}
	ivkURL.Methods = strings.Split(methods, ",")
	proxyFactory := extension.GetProxyFactory("")
	invoker := proxyFactory.GetInvoker(ivkURL)
	exporter.protocolExporter = extension.GetProtocol(ivkURL.Protocol).Export(invoker)
	metadataUrl = exporter.protocolExporter.GetInvoker().GetURL()
	logger.Infof("[Metadata Service] The MetadataService exports urls : %v ", exporter.protocolExporter.GetInvoker().GetURL())
	return nil
}

// UnExport will unExport the metadataService
func (exporter *ServiceExporter) UnExport() {
	exporter.protocolExporter.UnExport()
}
