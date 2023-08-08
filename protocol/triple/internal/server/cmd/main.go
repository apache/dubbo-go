package main

import (
	"context"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/greettriple"
	"dubbo.apache.org/dubbo-go/v3/provider"
	"errors"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	"io"
	"strings"
)

type GreetConnectServer struct {
}

func (srv *GreetConnectServer) Greet(ctx context.Context, req *greet.GreetRequest) (*greet.GreetResponse, error) {
	resp := &greet.GreetResponse{Greeting: "hello " + req.Name}
	return resp, nil
}

func (srv *GreetConnectServer) GreetStream(ctx context.Context, stream greettriple.GreetService_GreetStreamServer) error {
	for {
		req, err := stream.Recv()
		if err != nil {
			return fmt.Errorf("connect recv error: %s", err)
		}
		if err := stream.Send(&greet.GreetStreamResponse{Greeting: "hello " + req.Name}); err != nil {
			return fmt.Errorf("connect send error: %s", err)
		}
	}
}

func (srv *GreetConnectServer) GreetClientStream(ctx context.Context, stream greettriple.GreetService_GreetClientStreamServer) (*greet.GreetClientStreamResponse, error) {
	var reqs []string
	for stream.Recv() {
		reqs = append(reqs, stream.Msg().Name)
	}
	if stream.Err() != nil && !errors.Is(stream.Err(), io.EOF) {
		logger.Errorf("ClientStream unexpected err: %s", stream.Err())
	}
	resp := &greet.GreetClientStreamResponse{
		Greeting: "Hello" + strings.Join(reqs, ","),
	}

	return resp, nil
}

func (srv *GreetConnectServer) GreetServerStream(ctx context.Context, req *greet.GreetServerStreamRequest, stream greettriple.GreetService_GreetServerStreamServer) error {
	for i := 0; i < 5; i++ {
		if err := stream.Send(&greet.GreetServerStreamResponse{Greeting: "Hello" + req.Name}); err != nil {
			logger.Errorf("ServerStream unexpected err: %s", err)
		}
	}
	return nil
}

func main() {
	pro, err := provider.NewProvider()
	if err != nil {
		panic(err)
	}
	if err := greettriple.ProvideGreetServiceHandler(pro, &GreetConnectServer{}); err != nil {
		panic(err)
	}
	select {}
}
