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

package metadata

import (
	"context"
	"strconv"
	"strings"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	tripleapi "dubbo.apache.org/dubbo-go/v3/metadata/triple_api/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/base"
	"dubbo.apache.org/dubbo-go/v3/protocol/protocolwrapper"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

// version will be used by Version func
const (
	version  = constant.MetadataServiceV1Version
	allMatch = "*"
)

// MetadataService is used to define meta data related behaviors
// usually the implementation should be singleton
type MetadataService interface {
	// GetExportedURLs will get the target exported url in metadata, the url should be unique
	GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]*common.URL, error)
	// GetExportedServiceURLs will return exported service urls
	GetExportedServiceURLs() ([]*common.URL, error)
	// GetSubscribedURLs will get the exported urls in metadata
	GetSubscribedURLs() ([]*common.URL, error)
	Version() (string, error)
	// GetMetadataInfo will return metadata info
	GetMetadataInfo(revision string) (*info.MetadataInfo, error)
	// GetMetadataServiceURL will return the url of metadata service
	GetMetadataServiceURL() (*common.URL, error)
}

// DefaultMetadataService is store and query the metadata info in memory when each service registry
type DefaultMetadataService struct {
	metadataMap map[string]*info.MetadataInfo
	metadataUrl *common.URL
}

func (mts *DefaultMetadataService) setMetadataServiceURL(url *common.URL) {
	mts.metadataUrl = url
}

// GetExportedURLs get all exported urls
func (mts *DefaultMetadataService) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]*common.URL, error) {
	all, err := mts.GetExportedServiceURLs()
	if err != nil {
		return nil, err
	}
	urls := make([]*common.URL, 0)
	for _, url := range all {
		if (url.Interface() == serviceInterface || serviceInterface == allMatch) &&
			(url.Group() == group || group == allMatch) &&
			(url.Protocol == protocol || protocol == allMatch) &&
			(url.Version() == version || version == allMatch) {
			urls = append(urls, url)
		}
	}
	return urls, nil
}

// GetMetadataInfo can get metadata in memory
func (mts *DefaultMetadataService) GetMetadataInfo(revision string) (*info.MetadataInfo, error) {
	if revision == "" {
		return nil, nil
	}
	for _, metadataInfo := range mts.metadataMap {
		if metadataInfo.Revision == revision {
			return metadataInfo, nil
		}
	}
	logger.Warnf("metadata not found for revision: %s", revision)
	return nil, nil
}

// GetExportedServiceURLs get exported service urls
func (mts *DefaultMetadataService) GetExportedServiceURLs() ([]*common.URL, error) {
	urls := make([]*common.URL, 0)
	for _, metadataInfo := range mts.metadataMap {
		urls = append(urls, metadataInfo.GetExportedServiceURLs()...)
	}
	return urls, nil
}

// Version will return the version of metadata service
func (mts *DefaultMetadataService) Version() (string, error) {
	return version, nil
}

// GetMetadataServiceURL get url of MetadataService
func (mts *DefaultMetadataService) GetMetadataServiceURL() (*common.URL, error) {
	return mts.metadataUrl, nil
}

func (mts *DefaultMetadataService) GetSubscribedURLs() ([]*common.URL, error) {
	urls := make([]*common.URL, 0)
	for _, metadataInfo := range mts.metadataMap {
		urls = append(urls, metadataInfo.GetSubscribedURLs()...)
	}
	return urls, nil
}

// MethodMapper only for rename exported function, for example: rename the function GetMetadataInfo to getMetadataInfo
func (mts *DefaultMetadataService) MethodMapper() map[string]string {
	return map[string]string{
		"GetExportedURLs": "getExportedURLs",
		"GetMetadataInfo": "getMetadataInfo",
	}
}

// serviceExporter export MetadataService
type serviceExporter struct {
	opts             *Options
	service          MetadataService
	protocolExporter base.Exporter
	v2Exporter       base.Exporter
}

// Export will export the metadataService
func (e *serviceExporter) Export() error {
	var port string
	if e.opts.port == 0 {
		port = common.GetRandomPort("")
	} else {
		port = strconv.Itoa(e.opts.port)
	}
	if e.opts.protocol == constant.DefaultProtocol {
		err := e.exportDubbo(port)
		if err != nil {
			return err
		}
	} else {
		e.exportTripleV1(port)
		// v2 only supports triple protocol
		e.exportV2(port)
	}
	return nil
}

func (e *serviceExporter) exportDubbo(port string) error {
	version, _ := e.service.Version()
	ivkURL := common.NewURLWithOptions(
		common.WithPath(constant.MetadataServiceName),
		common.WithProtocol(constant.DefaultProtocol),
		common.WithPort(port),
		common.WithParamsValue(constant.GroupKey, e.opts.appName),
		common.WithParamsValue(constant.SerializationKey, constant.Hessian2Serialization),
		common.WithParamsValue(constant.ReleaseKey, constant.Version),
		common.WithParamsValue(constant.VersionKey, version),
		common.WithParamsValue(constant.InterfaceKey, constant.MetadataServiceName),
		common.WithParamsValue(constant.BeanNameKey, constant.SimpleMetadataServiceName),
		common.WithParamsValue(constant.MetadataTypeKey, e.opts.metadataType),
		common.WithParamsValue(constant.SideKey, constant.SideProvider),
	)
	methods, err := common.ServiceMap.Register(ivkURL.Interface(), ivkURL.Protocol, ivkURL.Group(), ivkURL.Version(), e.service)
	if err != nil {
		formatErr := perrors.Errorf("The service %v needExport the protocol %v error! Error message is %v.",
			ivkURL.Interface(), ivkURL.Protocol, err.Error())
		logger.Errorf(formatErr.Error())
		return formatErr
	}
	ivkURL.Methods = strings.Split(methods, ",")
	proxyFactory := extension.GetProxyFactory("")
	invoker := proxyFactory.GetInvoker(ivkURL)
	e.protocolExporter = extension.GetProtocol(ivkURL.Protocol).Export(invoker)
	e.service.(*DefaultMetadataService).setMetadataServiceURL(ivkURL)
	return nil
}

func (e *serviceExporter) exportTripleV1(port string) {
	version, _ := e.service.Version()
	svc := &MetadataServiceV1{delegate: e.service}
	ivkURL := common.NewURLWithOptions(
		common.WithPath(constant.MetadataServiceName),
		common.WithProtocol(constant.TriProtocol),
		common.WithPort(port),
		common.WithParamsValue(constant.GroupKey, e.opts.appName),
		common.WithParamsValue(constant.VersionKey, version),
		common.WithInterface(constant.MetadataServiceName),
		common.WithMethods(strings.Split("getMetadataInfo,GetMetadataInfo", ",")),
		common.WithParamsValue(constant.SerializationKey, constant.Hessian2Serialization),
		common.WithAttribute(constant.ServiceInfoKey, &MetadataService_ServiceInfo),
		common.WithAttribute(constant.RpcServiceKey, svc),
	)
	proxyFactory := extension.GetProxyFactory("")
	invoker := proxyFactory.GetInvoker(ivkURL)
	e.protocolExporter = extension.GetProtocol(protocolwrapper.FILTER).Export(invoker)
	e.service.(*DefaultMetadataService).setMetadataServiceURL(invoker.GetURL())
}

func (e *serviceExporter) exportV2(port string) {
	v2 := &MetadataServiceV2{delegate: e.service}
	// v2 only supports triple protocol
	ivkURL := common.NewURLWithOptions(
		common.WithPath(constant.MetadataServiceV2Name),
		common.WithProtocol(constant.TriProtocol),
		common.WithPort(port),
		common.WithParamsValue(constant.GroupKey, e.opts.appName),
		common.WithParamsValue(constant.VersionKey, "2.0.0"),
		common.WithInterface(constant.MetadataServiceV2Name),
		common.WithMethods(strings.Split("getMetadataInfo,GetMetadataInfo", ",")),
		common.WithAttribute(constant.ServiceInfoKey, &MetadataServiceV2_ServiceInfo),
		common.WithAttribute(constant.RpcServiceKey, v2),
	)
	proxyFactory := extension.GetProxyFactory("")
	invoker := proxyFactory.GetInvoker(ivkURL)
	e.v2Exporter = extension.GetProtocol(protocolwrapper.FILTER).Export(invoker)
	// do not set, because it will override MetadataService
	//exporter.metadataService.SetMetadataServiceURL(ivkURL)
}

// MetadataServiceV2Handler is an implementation of the org.apache.dubbo.metadata.MetadataServiceV2 service.
type MetadataServiceV2Handler interface {
	GetMetadataInfo(context.Context, *tripleapi.MetadataRequest) (*tripleapi.MetadataInfoV2, error)
}

type MetadataServiceV2 struct {
	delegate MetadataService
}

func (mtsV2 *MetadataServiceV2) GetMetadataInfo(ctx context.Context, req *tripleapi.MetadataRequest) (*tripleapi.MetadataInfoV2, error) {
	metadataInfo, err := mtsV2.delegate.GetMetadataInfo(req.GetRevision())
	if err != nil {
		return nil, err
	}
	return &tripleapi.MetadataInfoV2{
		App:      metadataInfo.App,
		Version:  metadataInfo.Revision,
		Services: convertV2(metadataInfo.Services),
	}, err
}

func convertV2(serviceInfos map[string]*info.ServiceInfo) map[string]*tripleapi.ServiceInfoV2 {
	serviceInfoV2s := make(map[string]*tripleapi.ServiceInfoV2, len(serviceInfos))
	for k, serviceInfo := range serviceInfos {
		serviceInfoV2 := &tripleapi.ServiceInfoV2{
			Name:     serviceInfo.Name,
			Group:    serviceInfo.Group,
			Version:  serviceInfo.Version,
			Protocol: serviceInfo.Protocol,
			Port:     0,
			Path:     serviceInfo.Path,
			Params:   serviceInfo.Params,
		}
		serviceInfoV2s[k] = serviceInfoV2
	}
	return serviceInfoV2s
}

var MetadataService_ServiceInfo = common.ServiceInfo{
	InterfaceName: "org.apache.dubbo.metadata.MetadataService",
	ServiceType:   (*MetadataServiceHandler)(nil),
	Methods: []common.MethodInfo{
		{
			Name: "getMetadataInfo",
			Type: constant.CallUnary,
			ReqInitFunc: func() any {
				return new(string)
			},
			MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
				revision := args[0].(*string)
				res, err := handler.(MetadataServiceHandler).GetMetadataInfo(ctx, *revision)
				return res, err
			},
		},
	},
}

var MetadataServiceV2_ServiceInfo = common.ServiceInfo{
	InterfaceName: "org.apache.dubbo.metadata.MetadataServiceV2",
	ServiceType:   (*MetadataServiceV2Handler)(nil),
	Methods: []common.MethodInfo{
		{
			Name: "GetMetadataInfo",
			Type: constant.CallUnary,
			ReqInitFunc: func() any {
				return new(tripleapi.MetadataRequest)
			},
			MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
				req := args[0].(*tripleapi.MetadataRequest)
				res, err := handler.(MetadataServiceV2Handler).GetMetadataInfo(ctx, req)
				if err != nil {
					return nil, err
				}
				return triple_protocol.NewResponse(res), nil
			},
		},
		{
			Name: "getMetadataInfo",
			Type: constant.CallUnary,
			ReqInitFunc: func() any {
				return new(tripleapi.MetadataRequest)
			},
			MethodFunc: func(ctx context.Context, args []any, handler any) (any, error) {
				req := args[0].(*tripleapi.MetadataRequest)
				res, err := handler.(MetadataServiceV2Handler).GetMetadataInfo(ctx, req)
				if err != nil {
					return nil, err
				}
				return triple_protocol.NewResponse(res), nil
			},
		},
	},
}

// MetadataServiceV1Handler
// 兼容 V1 接口定义
// 注意：V1 的方法签名与 V2 不同

type MetadataServiceHandler interface {
	GetMetadataInfo(ctx context.Context, revision string) (*info.MetadataInfo, error)
}

// MetadataServiceV1 的最小实现，用于导出 triple v1

type MetadataServiceV1 struct {
	delegate MetadataService
}

func (mtsV1 *MetadataServiceV1) GetMetadataInfo(ctx context.Context, revision string) (*info.MetadataInfo, error) {
	return mtsV1.delegate.GetMetadataInfo(revision)
}
