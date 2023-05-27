package main

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/internal/proto/greetconnect"
	"fmt"
)

type GreetConnectServer struct {
	greetconnect.GreetServiceProviderBase
}

func (srv *GreetConnectServer) Greet(ctx context.Context, c *connect.Request[greet.GreetRequest]) (*connect.Response[greet.GreetResponse], error) {
	resp := connect.NewResponse(&greet.GreetResponse{Greeting: "hello " + c.Msg.Name})
	return resp, nil
}

func (srv *GreetConnectServer) GreetStream(ctx context.Context, c *connect.BidiStream[greet.GreetStreamRequest, greet.GreetStreamResponse]) error {
	for {
		req, err := c.Receive()
		if err != nil {
			return fmt.Errorf("connect recv error: %s", err)
		}
		if err := c.Send(&greet.GreetStreamResponse{Greeting: "hello " + req.Name}); err != nil {
			return fmt.Errorf("connect send error: %s", err)
		}
	}
}

func main() {
	config.SetProviderService(&GreetConnectServer{})
	if err := config.Load(config.WithPath("./protocol/grpc_new/internal/server/cmd/dubbogo.yml")); err != nil {
		panic(err)
	}
	select {}
}
