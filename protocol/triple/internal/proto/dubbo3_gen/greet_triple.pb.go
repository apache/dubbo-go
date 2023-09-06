// Code generated by protoc-gen-go-triple. DO NOT EDIT.
// versions:
// - protoc-gen-go-triple v1.0.5
// - protoc             v3.20.3
// source: greet.proto

package greet

import (
	context "context"
	protocol "dubbo.apache.org/dubbo-go/v3/protocol"
	dubbo3 "dubbo.apache.org/dubbo-go/v3/protocol/dubbo3"
	invocation "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	fmt "fmt"
	grpc_go "github.com/dubbogo/grpc-go"
	codes "github.com/dubbogo/grpc-go/codes"
	metadata "github.com/dubbogo/grpc-go/metadata"
	status "github.com/dubbogo/grpc-go/status"
	common "github.com/dubbogo/triple/pkg/common"
	constant "github.com/dubbogo/triple/pkg/common/constant"
	triple "github.com/dubbogo/triple/pkg/triple"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
const _ = grpc_go.SupportPackageIsVersion7

// GreetServiceClient is the client API for GreetService service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type GreetServiceClient interface {
	Greet(ctx context.Context, in *proto.GreetRequest, opts ...grpc_go.CallOption) (*proto.GreetResponse, common.ErrorWithAttachment)
	GreetStream(ctx context.Context, opts ...grpc_go.CallOption) (GreetService_GreetStreamClient, error)
	GreetClientStream(ctx context.Context, opts ...grpc_go.CallOption) (GreetService_GreetClientStreamClient, error)
	GreetServerStream(ctx context.Context, in *proto.GreetServerStreamRequest, opts ...grpc_go.CallOption) (GreetService_GreetServerStreamClient, error)
}

type greetServiceClient struct {
	cc *triple.TripleConn
}

type GreetServiceClientImpl struct {
	Greet             func(ctx context.Context, in *proto.GreetRequest) (*proto.GreetResponse, error)
	GreetStream       func(ctx context.Context) (GreetService_GreetStreamClient, error)
	GreetClientStream func(ctx context.Context) (GreetService_GreetClientStreamClient, error)
	GreetServerStream func(ctx context.Context, in *proto.GreetServerStreamRequest) (GreetService_GreetServerStreamClient, error)
}

func (c *GreetServiceClientImpl) GetDubboStub(cc *triple.TripleConn) GreetServiceClient {
	return NewGreetServiceClient(cc)
}

func (c *GreetServiceClientImpl) XXX_InterfaceName() string {
	return "greet.GreetService"
}

func NewGreetServiceClient(cc *triple.TripleConn) GreetServiceClient {
	return &greetServiceClient{cc}
}

func (c *greetServiceClient) Greet(ctx context.Context, in *proto.GreetRequest, opts ...grpc_go.CallOption) (*proto.GreetResponse, common.ErrorWithAttachment) {
	out := new(proto.GreetResponse)
	interfaceKey := ctx.Value(constant.InterfaceKey).(string)
	return out, c.cc.Invoke(ctx, "/"+interfaceKey+"/Greet", in, out)
}

func (c *greetServiceClient) GreetStream(ctx context.Context, opts ...grpc_go.CallOption) (GreetService_GreetStreamClient, error) {
	interfaceKey := ctx.Value(constant.InterfaceKey).(string)
	stream, err := c.cc.NewStream(ctx, "/"+interfaceKey+"/GreetStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &greetServiceGreetStreamClient{stream}
	return x, nil
}

type GreetService_GreetStreamClient interface {
	Send(*proto.GreetStreamRequest) error
	Recv() (*proto.GreetStreamResponse, error)
	grpc_go.ClientStream
}

type greetServiceGreetStreamClient struct {
	grpc_go.ClientStream
}

func (x *greetServiceGreetStreamClient) Send(m *proto.GreetStreamRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *greetServiceGreetStreamClient) Recv() (*proto.GreetStreamResponse, error) {
	m := new(proto.GreetStreamResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *greetServiceClient) GreetClientStream(ctx context.Context, opts ...grpc_go.CallOption) (GreetService_GreetClientStreamClient, error) {
	interfaceKey := ctx.Value(constant.InterfaceKey).(string)
	stream, err := c.cc.NewStream(ctx, "/"+interfaceKey+"/GreetClientStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &greetServiceGreetClientStreamClient{stream}
	return x, nil
}

type GreetService_GreetClientStreamClient interface {
	Send(*proto.GreetClientStreamRequest) error
	CloseAndRecv() (*proto.GreetClientStreamResponse, error)
	grpc_go.ClientStream
}

type greetServiceGreetClientStreamClient struct {
	grpc_go.ClientStream
}

func (x *greetServiceGreetClientStreamClient) Send(m *proto.GreetClientStreamRequest) error {
	return x.ClientStream.SendMsg(m)
}

func (x *greetServiceGreetClientStreamClient) CloseAndRecv() (*proto.GreetClientStreamResponse, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(proto.GreetClientStreamResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *greetServiceClient) GreetServerStream(ctx context.Context, in *proto.GreetServerStreamRequest, opts ...grpc_go.CallOption) (GreetService_GreetServerStreamClient, error) {
	interfaceKey := ctx.Value(constant.InterfaceKey).(string)
	stream, err := c.cc.NewStream(ctx, "/"+interfaceKey+"/GreetServerStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &greetServiceGreetServerStreamClient{stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

type GreetService_GreetServerStreamClient interface {
	Recv() (*proto.GreetServerStreamResponse, error)
	grpc_go.ClientStream
}

type greetServiceGreetServerStreamClient struct {
	grpc_go.ClientStream
}

func (x *greetServiceGreetServerStreamClient) Recv() (*proto.GreetServerStreamResponse, error) {
	m := new(proto.GreetServerStreamResponse)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// GreetServiceServer is the server API for GreetService service.
// All implementations must embed UnimplementedGreetServiceServer
// for forward compatibility
type GreetServiceServer interface {
	Greet(context.Context, *proto.GreetRequest) (*proto.GreetResponse, error)
	GreetStream(GreetService_GreetStreamServer) error
	GreetClientStream(GreetService_GreetClientStreamServer) error
	GreetServerStream(*proto.GreetServerStreamRequest, GreetService_GreetServerStreamServer) error
	mustEmbedUnimplementedGreetServiceServer()
}

// UnimplementedGreetServiceServer must be embedded to have forward compatible implementations.
type UnimplementedGreetServiceServer struct {
	proxyImpl protocol.Invoker
}

func (UnimplementedGreetServiceServer) Greet(context.Context, *proto.GreetRequest) (*proto.GreetResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Greet not implemented")
}
func (UnimplementedGreetServiceServer) GreetStream(GreetService_GreetStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method GreetStream not implemented")
}
func (UnimplementedGreetServiceServer) GreetClientStream(GreetService_GreetClientStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method GreetClientStream not implemented")
}
func (UnimplementedGreetServiceServer) GreetServerStream(*proto.GreetServerStreamRequest, GreetService_GreetServerStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method GreetServerStream not implemented")
}
func (s *UnimplementedGreetServiceServer) XXX_SetProxyImpl(impl protocol.Invoker) {
	s.proxyImpl = impl
}

func (s *UnimplementedGreetServiceServer) XXX_GetProxyImpl() protocol.Invoker {
	return s.proxyImpl
}

func (s *UnimplementedGreetServiceServer) XXX_ServiceDesc() *grpc_go.ServiceDesc {
	return &GreetService_ServiceDesc
}
func (s *UnimplementedGreetServiceServer) XXX_InterfaceName() string {
	return "greet.GreetService"
}

func (UnimplementedGreetServiceServer) mustEmbedUnimplementedGreetServiceServer() {}

// UnsafeGreetServiceServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to GreetServiceServer will
// result in compilation errors.
type UnsafeGreetServiceServer interface {
	mustEmbedUnimplementedGreetServiceServer()
}

func RegisterGreetServiceServer(s grpc_go.ServiceRegistrar, srv GreetServiceServer) {
	s.RegisterService(&GreetService_ServiceDesc, srv)
}

func _GreetService_Greet_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc_go.UnaryServerInterceptor) (interface{}, error) {
	in := new(proto.GreetRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	base := srv.(dubbo3.Dubbo3GrpcService)
	args := []interface{}{}
	args = append(args, in)
	md, _ := metadata.FromIncomingContext(ctx)
	invAttachment := make(map[string]interface{}, len(md))
	for k, v := range md {
		invAttachment[k] = v
	}
	invo := invocation.NewRPCInvocation("Greet", args, invAttachment)
	if interceptor == nil {
		result := base.XXX_GetProxyImpl().Invoke(ctx, invo)
		return result, result.Error()
	}
	info := &grpc_go.UnaryServerInfo{
		Server:     srv,
		FullMethod: ctx.Value("XXX_TRIPLE_GO_INTERFACE_NAME").(string),
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		result := base.XXX_GetProxyImpl().Invoke(ctx, invo)
		return result, result.Error()
	}
	return interceptor(ctx, in, info, handler)
}

func _GreetService_GreetStream_Handler(srv interface{}, stream grpc_go.ServerStream) error {
	_, ok := srv.(dubbo3.Dubbo3GrpcService)
	invo := invocation.NewRPCInvocation("GreetStream", nil, nil)
	if !ok {
		fmt.Println(invo)
		return nil
	}
	return srv.(GreetServiceServer).GreetStream(&greetServiceGreetStreamServer{stream})
}

type GreetService_GreetStreamServer interface {
	Send(*proto.GreetStreamResponse) error
	Recv() (*proto.GreetStreamRequest, error)
	grpc_go.ServerStream
}

type greetServiceGreetStreamServer struct {
	grpc_go.ServerStream
}

func (x *greetServiceGreetStreamServer) Send(m *proto.GreetStreamResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *greetServiceGreetStreamServer) Recv() (*proto.GreetStreamRequest, error) {
	m := new(proto.GreetStreamRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _GreetService_GreetClientStream_Handler(srv interface{}, stream grpc_go.ServerStream) error {
	_, ok := srv.(dubbo3.Dubbo3GrpcService)
	invo := invocation.NewRPCInvocation("GreetClientStream", nil, nil)
	if !ok {
		fmt.Println(invo)
		return nil
	}
	return srv.(GreetServiceServer).GreetClientStream(&greetServiceGreetClientStreamServer{stream})
}

type GreetService_GreetClientStreamServer interface {
	SendAndClose(*proto.GreetClientStreamResponse) error
	Recv() (*proto.GreetClientStreamRequest, error)
	grpc_go.ServerStream
}

type greetServiceGreetClientStreamServer struct {
	grpc_go.ServerStream
}

func (x *greetServiceGreetClientStreamServer) SendAndClose(m *proto.GreetClientStreamResponse) error {
	return x.ServerStream.SendMsg(m)
}

func (x *greetServiceGreetClientStreamServer) Recv() (*proto.GreetClientStreamRequest, error) {
	m := new(proto.GreetClientStreamRequest)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _GreetService_GreetServerStream_Handler(srv interface{}, stream grpc_go.ServerStream) error {
	_, ok := srv.(dubbo3.Dubbo3GrpcService)
	invo := invocation.NewRPCInvocation("GreetServerStream", nil, nil)
	if !ok {
		fmt.Println(invo)
		return nil
	}
	m := new(proto.GreetServerStreamRequest)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(GreetServiceServer).GreetServerStream(m, &greetServiceGreetServerStreamServer{stream})
}

type GreetService_GreetServerStreamServer interface {
	Send(*proto.GreetServerStreamResponse) error
	grpc_go.ServerStream
}

type greetServiceGreetServerStreamServer struct {
	grpc_go.ServerStream
}

func (x *greetServiceGreetServerStreamServer) Send(m *proto.GreetServerStreamResponse) error {
	return x.ServerStream.SendMsg(m)
}

// GreetService_ServiceDesc is the grpc_go.ServiceDesc for GreetService service.
// It's only intended for direct use with grpc_go.RegisterService,
// and not to be introspected or modified (even as a copy)
var GreetService_ServiceDesc = grpc_go.ServiceDesc{
	ServiceName: "greet.GreetService",
	HandlerType: (*GreetServiceServer)(nil),
	Methods: []grpc_go.MethodDesc{
		{
			MethodName: "Greet",
			Handler:    _GreetService_Greet_Handler,
		},
	},
	Streams: []grpc_go.StreamDesc{
		{
			StreamName:    "GreetStream",
			Handler:       _GreetService_GreetStream_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "GreetClientStream",
			Handler:       _GreetService_GreetClientStream_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "GreetServerStream",
			Handler:       _GreetService_GreetServerStream_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "greet.proto",
}
