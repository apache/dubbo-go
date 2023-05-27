package main

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/internal/proto/greetconnect"
	"github.com/dubbogo/gost/log/logger"
)

var cli = new(greetconnect.GreetServiceClientImpl)

func main() {
	config.SetConsumerService(cli)
	if err := config.Load(config.WithPath("./protocol/grpc_new/internal/client/cmd/dubbogo.yml")); err != nil {
		panic(err)
	}

	logger.Info("start to test grpc_new")
	resp, err := cli.Greet(context.Background(), connect.NewRequest(&greet.GreetRequest{Name: "dubbo"}))
	if err != nil {
		logger.Error(err)
		return
	}
	logger.Infof("grpc_new unary call resp: %s", resp.Msg.Greeting)
}
