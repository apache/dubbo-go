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

package server

import (
	"context"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	triple_api "dubbo.apache.org/dubbo-go/v3/metadata/triple_api/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

// MetadataServiceV2Handler is an implementation of the org.apache.dubbo.metadata.MetadataServiceV2 service.
type MetadataServiceV2Handler interface {
	GetMetadataInfo(context.Context, *triple_api.MetadataRequest) (*triple_api.MetadataInfoV2, error)
}

var MetadataServiceV2_ServiceInfo = ServiceInfo{
	InterfaceName: "org.apache.dubbo.metadata.MetadataServiceV2",
	ServiceType:   (*MetadataServiceV2Handler)(nil),
	Methods: []MethodInfo{
		{
			Name: "GetMetadataInfo",
			Type: constant.CallUnary,
			ReqInitFunc: func() interface{} {
				return new(triple_api.MetadataRequest)
			},
			MethodFunc: func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error) {
				req := args[0].(*triple_api.MetadataRequest)
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
			ReqInitFunc: func() interface{} {
				return new(triple_api.MetadataRequest)
			},
			MethodFunc: func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error) {
				req := args[0].(*triple_api.MetadataRequest)
				res, err := handler.(MetadataServiceV2Handler).GetMetadataInfo(ctx, req)
				if err != nil {
					return nil, err
				}
				return triple_protocol.NewResponse(res), nil
			},
		},
	},
}

type MetadataServiceHandler interface {
	GetMetadataInfo(ctx context.Context, revision string) (*triple_api.MetadataInfo, error)
}

var MetadataService_ServiceInfo = ServiceInfo{
	InterfaceName: "org.apache.dubbo.metadata.MetadataService",
	ServiceType:   (*MetadataServiceHandler)(nil),
	Methods: []MethodInfo{
		{
			Name: "getMetadataInfo",
			Type: constant.CallUnary,
			ReqInitFunc: func() interface{} {
				return new(string)
			},
			MethodFunc: func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error) {
				revision := args[0].(*string)
				res, err := handler.(MetadataServiceHandler).GetMetadataInfo(ctx, *revision)
				return res, err
			},
		},
	},
}
