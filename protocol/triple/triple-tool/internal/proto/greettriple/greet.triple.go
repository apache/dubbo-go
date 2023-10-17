// Code generated by protoc-gen-triple. DO NOT EDIT.
//
// Source: internal/proto/greet.proto
package greet

import (
	context "context"
	http "net/http"
)

import (
	client "dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	triple_protocol "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"dubbo.apache.org/dubbo-go/v3/server"
)

import (
	proto "dubbo.apache.org/dubbo-go/v3/triple-tool/internal/proto"
)

// This is a compile-time assertion to ensure that this generated file and the Triple package
// are compatible. If you get a compiler error that this constant is not defined, this code was
// generated with a version ofTtriple newer than the one compiled into your binary. You can fix the
// problem by either regenerating this code with an older version of Triple or updating the Triple
// version compiled into your binary.
const _ = triple_protocol.IsAtLeastVersion0_1_0

const (
	// GreetServiceName is the fully-qualified name of the GreetService service.
	GreetServiceName = "greet.GreetService"
)

// These constants are the fully-qualified names of the RPCs defined in this package. They're
// exposed at runtime as Spec.Procedure and as the final two segments of the HTTP route.
//
// Note that these are different from the fully-qualified method names used by
// google.golang.org/protobuf/reflect/protoreflect. To convert from these constants to
// reflection-formatted method names, remove the leading slash and convert the remaining slash to a
// period.
const (
	// GreetServiceGreetProcedure is the fully-qualified name of the GreetService's Greet RPC.
	GreetServiceGreetProcedure = "/greet.GreetService/Greet"
	// GreetServiceGreetStreamProcedure is the fully-qualified name of the GreetService's GreetStream RPC.
	GreetServiceGreetStreamProcedure = "/greet.GreetService/GreetStream"
	// GreetServiceGreetClientStreamProcedure is the fully-qualified name of the GreetService's GreetClientStream RPC.
	GreetServiceGreetClientStreamProcedure = "/greet.GreetService/GreetClientStream"
	// GreetServiceGreetServerStreamProcedure is the fully-qualified name of the GreetService's GreetServerStream RPC.
	GreetServiceGreetServerStreamProcedure = "/greet.GreetService/GreetServerStream"
)

//GreetServiceClient is a client for the greet.GreetService service.
type GreetService interface {
	Greet(ctx context.Context, req *proto.GreetRequest, opt ...client.CallOption) (*proto.GreetResponse, error)

	GreetStream(ctx context.Context, opt ...client.CallOption) (GreetService_GreetStreamClient, error)

	GreetClientStream(ctx context.Context, opt ...client.CallOption) (GreetService_GreetClientStreamClient, error)

	GreetServerStream(ctx context.Context, req *proto.GreetServerStreamRequest, opt ...client.CallOption) (GreetService_GreetServerStreamClient, error)
}

// NewGreetService constructs a client for the greet.GreetService service. By default, it uses
// the Connect protocol with the binary Protobuf Codec, asks for gzipped responses, and sends
// uncompressed requests. To use the gRPC or gRPC-Web protocols, supply the connect.WithGRPC() or
// connect.WithGRPCWeb() options.
//
// The URL supplied here should be the base URL for the Triple server (for example,
// http://api.acme.com or https://acme.com/grpc).
func NewGreetService(cli *client.Client) (GreetService, error) {
	if err := cli.Init(&GreetService_ClientInfo); err != nil {
		return nil, err
	}
	return &GreetServiceImpl{
		cli: cli,
	}, nil
}

func SetConsumerService(srv common.RPCService) {
	config.SetClientInfoService(&GreetService_ClientInfo, srv)
}

// GreetServiceClientImpl implements GreetServiceClient.
type GreetServiceImpl struct {
	cli *client.Client
}

func (c *GreetServiceImpl) Greet(ctx context.Context, req *proto.GreetRequest, opts ...client.CallOption) (*proto.GreetResponse, error) {
	resp := new(proto.GreetResponse)
	if err := c.cli.CallUnary(ctx, req, resp, "greet.GreetService", "Greet", opts...); err != nil {
		return nil, err
	}
	return resp, nil
}

func (c *GreetServiceImpl) GreetStream(ctx context.Context, opts ...client.CallOption) (GreetService_GreetStreamClient, error) {
	stream, err := c.cli.CallBidiStream(ctx, "greet.GreetService", "GreetStream", opts...)
	if err != nil {
		return nil, err
	}
	rawStream := stream.(*triple_protocol.BidiStreamForClient)
	return &greetServiceGreetStreamClient{rawStream}, nil
}

func (c *GreetServiceImpl) GreetClientStream(ctx context.Context, opts ...client.CallOption) (GreetService_GreetClientStreamClient, error) {
	stream, err := c.cli.CallClientStream(ctx, "greet.GreetService", "GreetClientStream", opts...)
	if err != nil {
		return nil, err
	}
	rawStream := stream.(*triple_protocol.ClientStreamForClient)
	return &greetServiceGreetClientStreamClient{rawStream}, nil
}

func (c *GreetServiceImpl) GreetServerStream(ctx context.Context, req *proto.GreetServerStreamRequest, opts ...client.CallOption) (GreetService_GreetServerStreamClient, error) {
	stream, err := c.cli.CallServerStream(ctx, req, "greet.GreetService", "GreetServerStream", opts...)
	if err != nil {
		return nil, err
	}
	rawStream := stream.(*triple_protocol.ServerStreamForClient)
	return &greetServiceGreetServerStreamClient{rawStream}, nil
}

type GreetService_GreetClient interface {
	Recv() bool
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	Msg() *proto.GreetResponse
	Err() error
	Conn() (triple_protocol.StreamingClientConn, error)
	Close() error
}

type greetServiceGreetClient struct {
	*triple_protocol.ServerStreamForClient
}

func (cli *greetServiceGreetClient) Recv() bool {
	msg := new(proto.GreetResponse)
	return cli.ServerStreamForClient.Receive(msg)
}

func (cli *greetServiceGreetClient) Msg() *proto.GreetResponse {
	msg := cli.ServerStreamForClient.Msg()
	if msg == nil {
		return new(proto.GreetResponse)
	}
	return msg.(*proto.GreetResponse)
}

func (cli *greetServiceGreetClient) Conn() (triple_protocol.StreamingClientConn, error) {
	return cli.ServerStreamForClient.Conn()
}

type GreetService_GreetStreamClient interface {
	Spec() triple_protocol.Spec
	Peer() triple_protocol.Peer
	Send(*proto.GreetStreamRequest) error
	RequestHeader() http.Header
	CloseRequest() error
	Recv() (*proto.GreetStreamResponse, error)
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	CloseResponse() error
}

type greetServiceGreetStreamClient struct {
	*triple_protocol.BidiStreamForClient
}

func (cli *greetServiceGreetStreamClient) Send(msg *proto.GreetStreamRequest) error {
	return cli.BidiStreamForClient.Send(msg)
}

func (cli *greetServiceGreetStreamClient) Recv() (*proto.GreetStreamResponse, error) {
	msg := new(proto.GreetStreamResponse)
	if err := cli.BidiStreamForClient.Receive(msg); err != nil {
		return nil, err
	}
	return msg, nil
}

type GreetService_GreetClientStreamClient interface {
	Spec() triple_protocol.Spec
	Peer() triple_protocol.Peer
	Send(*proto.GreetClientStreamRequest) error
	RequestHeader() http.Header
	CloseAndRecv() (*proto.GreetClientStreamResponse, error)
	Conn() (triple_protocol.StreamingClientConn, error)
}

type greetServiceGreetClientStreamClient struct {
	*triple_protocol.ClientStreamForClient
}

func (cli *greetServiceGreetClientStreamClient) Send(msg *proto.GreetClientStreamRequest) error {
	return cli.ClientStreamForClient.Send(msg)
}

func (cli *greetServiceGreetClientStreamClient) CloseAndRecv() (*proto.GreetClientStreamResponse, error) {
	msg := new(proto.GreetClientStreamResponse)
	resp := triple_protocol.NewResponse(msg)
	if err := cli.ClientStreamForClient.CloseAndReceive(resp); err != nil {
		return nil, err
	}
	return msg, nil
}

func (cli *greetServiceGreetClientStreamClient) Conn() (triple_protocol.StreamingClientConn, error) {
	return cli.ClientStreamForClient.Conn()
}

type GreetService_GreetServerStreamClient interface {
	Recv() bool
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	Msg() *proto.GreetServerStreamResponse
	Err() error
	Conn() (triple_protocol.StreamingClientConn, error)
	Close() error
}

type greetServiceGreetServerStreamClient struct {
	*triple_protocol.ServerStreamForClient
}

func (cli *greetServiceGreetServerStreamClient) Recv() bool {
	msg := new(proto.GreetServerStreamResponse)
	return cli.ServerStreamForClient.Receive(msg)
}

func (cli *greetServiceGreetServerStreamClient) Msg() *proto.GreetServerStreamResponse {
	msg := cli.ServerStreamForClient.Msg()
	if msg == nil {
		return new(proto.GreetServerStreamResponse)
	}
	return msg.(*proto.GreetServerStreamResponse)
}

func (cli *greetServiceGreetServerStreamClient) Conn() (triple_protocol.StreamingClientConn, error) {
	return cli.ServerStreamForClient.Conn()
}

var GreetService_ClientInfo = client.ClientInfo{
	InterfaceName: "greet.GreetService",
	MethodNames:   []string{"Greet", "GreetStream", "GreetClientStream", "GreetServerStream"},
	ClientInjectFunc: func(dubboCliRaw interface{}, cli *client.Client) {
		dubboCli := dubboCliRaw.(GreetServiceImpl)
		dubboCli.cli = cli
	},
}

// GreetServiceHandler is an implementation of the greet.GreetService service.
type GreetServiceHandler interface {
	Greet(context.Context, *proto.GreetRequest) (*proto.GreetResponse, error)

	GreetStream(context.Context, GreetService_GreetStreamServer) error

	GreetClientStream(context.Context, GreetService_GreetClientStreamServer) (*proto.GreetClientStreamResponse, error)

	GreetServerStream(context.Context, *proto.GreetServerStreamRequest, GreetService_GreetServerStreamServer) error
}

func RegisterGreetServiceHandler(srv *server.Server, hdlr GreetServiceHandler, opts ...server.ServiceOption) error {
	return srv.Register(hdlr, &GreetService_ServiceInfo, opts...)
}

type GreetService_GreetServer interface {
	Send(*proto.GreetResponse) error
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	Conn() triple_protocol.StreamingHandlerConn
}

type GreetServiceGreetServer struct {
	*triple_protocol.ServerStream
}

func (g *GreetServiceGreetServer) Send(msg *proto.GreetResponse) error {
	return g.ServerStream.Send(msg)
}

type GreetService_GreetStreamServer interface {
	Send(*proto.GreetStreamResponse) error
	Recv() (*proto.GreetStreamRequest, error)
	Spec() triple_protocol.Spec
	Peer() triple_protocol.Peer
	RequestHeader() http.Header
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	Conn() triple_protocol.StreamingHandlerConn
}

type GreetServiceGreetStreamServer struct {
	*triple_protocol.BidiStream
}

func (srv *GreetServiceGreetStreamServer) Send(msg *proto.GreetStreamResponse) error {
	return srv.BidiStream.Send(msg)
}

func (srv GreetServiceGreetStreamServer) Recv() (*proto.GreetStreamRequest, error) {
	msg := new(proto.GreetStreamRequest)
	if err := srv.BidiStream.Receive(msg); err != nil {
		return nil, err
	}
	return msg, nil
}

type GreetService_GreetClientStreamServer interface {
	Spec() triple_protocol.Spec
	Peer() triple_protocol.Peer
	Recv() bool
	RequestHeader() http.Header
	Msg() *proto.GreetClientStreamRequest
	Err() error
	Conn() triple_protocol.StreamingHandlerConn
}

type GreetServiceGreetClientStreamServer struct {
	*triple_protocol.ClientStream
}

func (srv *GreetServiceGreetClientStreamServer) Recv() bool {
	msg := new(proto.GreetClientStreamRequest)
	return srv.ClientStream.Receive(msg)
}

func (srv *GreetServiceGreetClientStreamServer) Msg() *proto.GreetClientStreamRequest {
	msgRaw := srv.ClientStream.Msg()
	if msgRaw == nil {
		return new(proto.GreetClientStreamRequest)
	}
	return msgRaw.(*proto.GreetClientStreamRequest)
}

type GreetService_GreetServerStreamServer interface {
	Send(*proto.GreetServerStreamResponse) error
	ResponseHeader() http.Header
	ResponseTrailer() http.Header
	Conn() triple_protocol.StreamingHandlerConn
}

type GreetServiceGreetServerStreamServer struct {
	*triple_protocol.ServerStream
}

func (g *GreetServiceGreetServerStreamServer) Send(msg *proto.GreetServerStreamResponse) error {
	return g.ServerStream.Send(msg)
}

var GreetService_ServiceInfo = server.ServiceInfo{
	InterfaceName: "greet.GreetService",
	ServiceType:   (*GreetServiceHandler)(nil),
	Methods: []server.MethodInfo{
		{
			Name: "Greet",
			Type: constant.CallUnary,
			ReqInitFunc: func() interface{} {
				return new(proto.GreetRequest)
			},
			MethodFunc: func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error) {
				req := args[0].(*proto.GreetRequest)
				res, err := handler.(GreetServiceHandler).Greet(ctx, req)
				if err != nil {
					return nil, err
				}
				return triple_protocol.NewResponse(res), nil
			},
		},
		{
			Name: "GreetStream",
			Type: constant.CallBidiStream,
			StreamInitFunc: func(baseStream interface{}) interface{} {
				return &GreetServiceGreetStreamServer{baseStream.(*triple_protocol.BidiStream)}
			},
			MethodFunc: func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error) {
				stream := args[0].(GreetService_GreetStreamServer)
				if err := handler.(GreetServiceHandler).GreetStream(ctx, stream); err != nil {
					return nil, err
				}
				return nil, nil
			},
		},
		{
			Name: "GreetClientStream",
			Type: constant.CallClientStream,
			StreamInitFunc: func(baseStream interface{}) interface{} {
				return &GreetServiceGreetClientStreamServer{baseStream.(*triple_protocol.ClientStream)}
			},
			MethodFunc: func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error) {
				stream := args[0].(GreetService_GreetClientStreamServer)
				res, err := handler.(GreetServiceHandler).GreetClientStream(ctx, stream)
				if err != nil {
					return nil, err
				}
				return triple_protocol.NewResponse(res), nil
			},
		},
		{
			Name: "GreetServerStream",
			Type: constant.CallServerStream,
			ReqInitFunc: func() interface{} {
				return new(proto.GreetServerStreamRequest)
			},
			StreamInitFunc: func(baseStream interface{}) interface{} {
				return &GreetServiceGreetServerStreamServer{baseStream.(*triple_protocol.ServerStream)}
			},
			MethodFunc: func(ctx context.Context, args []interface{}, handler interface{}) (interface{}, error) {
				req := args[0].(*proto.GreetServerStreamRequest)
				stream := args[1].(GreetService_GreetServerStreamServer)
				if err := handler.(GreetServiceHandler).GreetServerStream(ctx, req, stream); err != nil {
					return nil, err
				}
				return nil, nil
			},
		},
	},
}
