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

package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/protobuf/ptypes"
	"google.golang.org/protobuf/types/known/structpb"
	"net"

	v3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"

	"github.com/dubbogo/gost/log/logger"
	envoy_config_core_v3 "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/type/v3"
	"github.com/golang/protobuf/jsonpb"
	"github.com/golang/protobuf/proto"
	_ "github.com/golang/protobuf/ptypes"
	_ "github.com/golang/protobuf/ptypes/any"

	_ "github.com/envoyproxy/go-control-plane/envoy/admin/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/cors/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/grpc_stats/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/jwt_authn/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"

	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/http_inspector/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/original_dst/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/listener/tls_inspector/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/rbac/v3"
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/upstreams/http/v3"
)

func ConvertAddress(xdsAddress *envoy_config_core_v3.Address) net.Addr {
	if xdsAddress == nil {
		return nil
	}
	var address string
	if addr, ok := xdsAddress.GetAddress().(*envoy_config_core_v3.Address_SocketAddress); ok {
		if xdsPort, ok := addr.SocketAddress.GetPortSpecifier().(*envoy_config_core_v3.SocketAddress_PortValue); ok {
			address = fmt.Sprintf("%s:%d", addr.SocketAddress.GetAddress(), xdsPort.PortValue)
		} else {
			logger.Warnf("only port value supported")
			return nil
		}
	} else {
		logger.Errorf("only SocketAddress supported")
		return nil
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		logger.Errorf("Invalid address: %v", err)
		return nil
	}
	return tcpAddr
}

func GetJsonString(msg proto.Message) string {

	m := jsonpb.Marshaler{
		OrigName: true,
		Indent:   " ",
	}
	value, err := ptypes.MarshalAny(msg)
	if err != nil {
		logger.Errorf("to any err:%v", err)
	}

	str, err := m.MarshalToString(value)
	if err != nil {
		logger.Errorf("to json string err:%v", err)
		return fmt.Sprintf("%v", msg)
	}
	return str
}

func ConvertResponseToString(discoveryResponse *v3.DiscoveryResponse) string {
	// 创建 jsonpb.Marshaler 对象
	marshaler := &jsonpb.Marshaler{Indent: "  "}

	// 创建缓冲区来保存 JSON 字符串
	var buffer bytes.Buffer

	// 将 DiscoveryResponse 转换为 JSON 字符串
	if err := marshaler.Marshal(&buffer, discoveryResponse); err != nil {
		logger.Errorf("failed to marshal DiscoveryResponse to JSON: %v", err)
	}
	return buffer.String()
}

func ConvertJsonString(data interface{}) string {
	//if bytes, err := json.Marshal(data); err == nil {
	//	return string(bytes)
	//}
	if bytes, err := json.MarshalIndent(data, "", "  "); err == nil {
		return string(bytes)
	}
	return ""
}

func ConvertTypesStruct(s *structpb.Struct) map[string]string {
	if s == nil {
		return nil
	}
	meta := make(map[string]string, len(s.GetFields()))
	for key, value := range s.GetFields() {
		meta[key] = value.String()
	}
	return meta
}

func ConvertAttachmentsToMap(attachments map[string]interface{}) map[string]string {
	dataMap := make(map[string]string, 0)
	for k, attachment := range attachments {
		if v, ok := attachment.([]string); ok {
			dataMap[k] = v[0]
		}
		if v, ok := attachment.(string); ok {
			dataMap[k] = v
		}
	}
	return dataMap
}
