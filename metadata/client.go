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
	"encoding/json"
	"reflect"
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
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

const defaultTimeout = "5s" // s

func GetMetadataFromMetadataReport(revision string, instance registry.ServiceInstance) (*info.MetadataInfo, error) {
	report := GetMetadataReport()
	if report == nil {
		return nil, perrors.New("no metadata report instance found,please check ")
	}
	return report.GetAppMetadata(instance.GetServiceName(), revision)
}

func GetMetadataFromRpc(revision string, instance registry.ServiceInstance) (*info.MetadataInfo, error) {
	url := buildStandardMetadataServiceURL(instance)
	url.SetParam(constant.TimeoutKey, defaultTimeout)
	p := extension.GetProtocol(url.Protocol)
	invoker := p.Refer(url)
	if invoker == nil { // can't connect instance
		return nil, perrors.New("can not connect to remote metadata service host: " + url.Ip)
	}
	var remoteService remoteMetadataService
	if instance.GetMetadata()[constant.MetadataVersion] == constant.MetadataServiceV2Version {
		remoteService = &triMetadataServiceV2{invoker: invoker}
	} else {
		remoteService = &remoteMetadataServiceV1{invoker: invoker}
	}
	return remoteService.getMetadataInfo(context.Background(), revision)
}

type remoteMetadataService interface {
	getMetadataInfo(context context.Context, revision string) (*info.MetadataInfo, error)
}

type triMetadataServiceV2 struct {
	invoker protocol.Invoker
}

func (m *triMetadataServiceV2) getMetadataInfo(ctx context.Context, revision string) (*info.MetadataInfo, error) {
	const methodName = "GetMetadataInfo"
	req := &tripleapi.MetadataRequest{Revision: revision}
	metadataInfo := &tripleapi.MetadataInfoV2{}
	inv, _ := generateInvocation(m.invoker.GetURL(), methodName, req, metadataInfo, constant.CallUnary)
	res := m.invoker.Invoke(context.Background(), inv)
	if res.Error() != nil {
		logger.Errorf("could not get the metadata info from remote provider: %v", res.Error())
		return nil, res.Error()
	}
	return convertMetadataInfoV2(metadataInfo), nil
}

func convertMetadataInfoV2(v2 *tripleapi.MetadataInfoV2) *info.MetadataInfo {
	infos := make(map[string]*info.ServiceInfo, 0)
	for k, v := range v2.Services {
		serviceInfo := &info.ServiceInfo{
			Name:     v.Name,
			Group:    v.Group,
			Version:  v.Version,
			Protocol: v.Protocol,
			Path:     v.Path,
			Params:   v.Params,
		}
		infos[k] = serviceInfo
	}

	metadataInfo := &info.MetadataInfo{
		App:      v2.App,
		Revision: v2.Version,
		Services: infos,
	}
	return metadataInfo
}

//func convertMetadataInfo(v1 *tripleapi.MetadataInfo) *info.MetadataInfo {
//	infos := make(map[string]*info.ServiceInfo, len(v1.Services))
//	for k, v := range v1.Services {
//		serviceInfo := &info.ServiceInfo{
//			Name:     v.Name,
//			Group:    v.Group,
//			Version:  v.Version,
//			Protocol: v.Protocol,
//			Path:     v.Path,
//			Params:   v.Params,
//		}
//		infos[k] = serviceInfo
//	}
//
//	metadataInfo := &info.MetadataInfo{
//		App:      v1.App,
//		Revision: v1.Version,
//		Services: infos,
//	}
//	return metadataInfo
//}

func generateInvocation(u *common.URL, methodName string, req interface{}, resp interface{}, callType string) (protocol.Invocation, error) {
	var inv *invocation.RPCInvocation
	if u.Protocol == constant.TriProtocol {
		var paramsRawVals []interface{}
		paramsRawVals = append(paramsRawVals, req)
		if resp != nil {
			paramsRawVals = append(paramsRawVals, resp)
		}
		inv = invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName(methodName),
			invocation.WithAttachment(constant.TimeoutKey, "5000"),
			invocation.WithAttachment(constant.RetriesKey, "2"),
			invocation.WithArguments([]interface{}{req}),
			invocation.WithReply(resp),
			invocation.WithParameterRawValues(paramsRawVals),
		)
		inv.SetAttribute(constant.CallTypeKey, callType)
	} else {
		rV := reflect.ValueOf(req)
		inv = invocation.NewRPCInvocationWithOptions(
			invocation.WithMethodName(methodName),
			invocation.WithArguments([]interface{}{rV.Interface()}),
			invocation.WithReply(resp),
			invocation.WithAttachments(map[string]interface{}{constant.AsyncKey: "false"}),
			invocation.WithParameterValues([]reflect.Value{rV}))
	}

	return inv, nil
}

type remoteMetadataServiceV1 struct {
	invoker protocol.Invoker
}

func (m *remoteMetadataServiceV1) getMetadataInfo(ctx context.Context, revision string) (*info.MetadataInfo, error) {
	const methodName = "getMetadataInfo"
	metadataInfo := &info.MetadataInfo{}
	inv, _ := generateInvocation(m.invoker.GetURL(), methodName, revision, metadataInfo, constant.CallUnary)
	res := m.invoker.Invoke(context.Background(), inv)
	if res.Error() != nil {
		logger.Errorf("could not get the metadata info from remote provider: %v", res.Error())
		return nil, res.Error()
	}
	if metadataInfo.Services == nil {
		metadataInfo = res.Result().(*info.MetadataInfo)
	}
	return metadataInfo, nil
}

// buildStandardMetadataServiceURL will use standard format to build the metadata service url.
func buildStandardMetadataServiceURL(ins registry.ServiceInstance) *common.URL {
	ps := getMetadataServiceUrlParams(ins)
	if ps[constant.ProtocolKey] == "" {
		return nil
	}

	sn := ins.GetServiceName()
	host := ins.GetHost()

	metaV := ins.GetMetadata()[constant.MetadataVersion]
	protocol := ps[constant.ProtocolKey]
	if metaV == constant.MetadataServiceV2Version {
		protocol = constant.TriProtocol
	}

	convertedParams := make(map[string][]string, len(ps))
	for k, v := range ps {
		convertedParams[k] = []string{v}
	}
	u := common.NewURLWithOptions(common.WithIp(host),
		common.WithPath(constant.MetadataServiceName),
		common.WithProtocol(protocol),
		common.WithPort(ps[constant.PortKey]),
		common.WithParams(convertedParams),
		common.WithParamsValue(constant.GroupKey, sn),
		common.WithParamsValue(constant.InterfaceKey, constant.MetadataServiceName))

	if protocol == constant.TriProtocol {
		u.SetAttribute(constant.ClientInfoKey, "info")
		u.Methods = []string{"GetMetadataInfo", "getMetadataInfo"}
		if metaV == constant.MetadataServiceV2Version {
			u.Path = constant.MetadataServiceV2Name
			u.SetParam(constant.VersionKey, metaV)
			u.SetParam(constant.InterfaceKey, constant.MetadataServiceV2Name)
			u.DelParam(constant.SerializationKey)
		} else {
			u.SetParam(constant.SerializationKey, constant.Hessian2Serialization)
		}
	}

	return u
}

// getMetadataServiceUrlParams this will convertV2 the metadata service url parameters to map structure
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
