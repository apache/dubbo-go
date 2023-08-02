package main

//
//import (
//	"context"
//	"dubbo.apache.org/dubbo-go/v3/config"
//	_ "dubbo.apache.org/dubbo-go/v3/imports"
//	greet "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/internal/proto"
//	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/internal/proto/greetconnect"
//	"errors"
//	"fmt"
//	"github.com/dubbogo/gost/log/logger"
//	"io"
//	"strings"
//)
//
//type GreetConnectServer struct {
//	greetconnect.GreetServiceProviderBase
//}
//
//func (srv *GreetConnectServer) Greet(ctx context.Context, req *triple.Request[greet.GreetRequest]) (*triple.Response[greet.GreetResponse], error) {
//	resp := triple.NewResponse(&greet.GreetResponse{Greeting: "hello " + req.Msg.Name})
//	return resp, nil
//}
//
//func (srv *GreetConnectServer) GreetStream(ctx context.Context, stream *triple.BidiStream[greet.GreetStreamRequest, greet.GreetStreamResponse]) error {
//	for {
//		req, err := stream.Receive()
//		if err != nil {
//			return fmt.Errorf("connect recv error: %s", err)
//		}
//		if err := stream.Send(&greet.GreetStreamResponse{Greeting: "hello " + req.Name}); err != nil {
//			return fmt.Errorf("connect send error: %s", err)
//		}
//	}
//}
//
//func (srv *GreetConnectServer) GreetClientStream(ctx context.Context, stream *triple.ClientStream[greet.GreetClientStreamRequest]) (*triple.Response[greet.GreetClientStreamResponse], error) {
//	var reqs []string
//	for stream.Receive() {
//		reqs = append(reqs, stream.Msg().Name)
//	}
//	if stream.Err() != nil && !errors.Is(stream.Err(), io.EOF) {
//		logger.Errorf("ClientStream unexpected err: %s", stream.Err())
//	}
//	resp := triple.NewResponse(
//		&greet.GreetClientStreamResponse{
//			Greeting: "Hello" + strings.Join(reqs, ","),
//		},
//	)
//	return resp, nil
//}
//
//func (srv *GreetConnectServer) GreetServerStream(ctx context.Context, req *triple.Request[greet.GreetServerStreamRequest], stream *triple.ServerStream[greet.GreetServerStreamResponse]) error {
//	for i := 0; i < 5; i++ {
//		if err := stream.Send(&greet.GreetServerStreamResponse{Greeting: "Hello" + req.Msg.Name}); err != nil {
//			logger.Errorf("ServerStream unexpected err: %s", err)
//		}
//	}
//	return nil
//}
//
//func main() {
//	config.SetProviderService(&GreetConnectServer{})
//	if err := config.Load(config.WithPath("./protocol/grpc_new/internal/server/cmd/dubbogo.yml")); err != nil {
//		panic(err)
//	}
//	select {}
//}
