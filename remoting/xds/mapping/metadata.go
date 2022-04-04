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

package mapping

import (
	structpb "github.com/golang/protobuf/ptypes/struct"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

// GetDubboGoMetadata create metadata of dubbo-go-app and register to istiod
func GetDubboGoMetadata(dubboGoMetadata string) *structpb.Struct {
	return &structpb.Struct{
		Fields: map[string]*structpb.Value{
			constant.XDSMetadataClusterIDKey: {
				// Set cluster id to Kubernetes to ensure dubbo-go's xds client can get service
				// istiod.istio-system.svc.cluster.local's
				// pods ip from istiod by eds, to call no-endpoint port of istio like 8080
				Kind: &structpb.Value_StringValue{StringValue: constant.XDSMetadataDefaultDomainName},
			},
			constant.XDSMetadataLabelsKey: {
				Kind: &structpb.Value_StructValue{StructValue: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						constant.XDSMetadataDubboGoMapperKey: {
							Kind: &structpb.Value_StringValue{StringValue: dubboGoMetadata},
						},
					},
				}},
			},
		},
	}
}
