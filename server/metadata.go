package server

import (
	"context"
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
