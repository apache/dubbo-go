package main

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/config"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/internal/proto/greetconnect"
	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/triple"
	"github.com/dubbogo/gost/log/logger"
)

var cli = new(greetconnect.GreetServiceClientImpl)

func main() {
	config.SetConsumerService(cli)
	if err := config.Load(config.WithPath("./protocol/grpc_new/internal/client/cmd/dubbogo.yml")); err != nil {
		panic(err)
	}

	if err := testUnary(cli); err != nil {
		logger.Error(err)
		return
	}

	if err := testBidiStream(cli); err != nil {
		logger.Error(err)
		return
	}

	if err := testClientStream(cli); err != nil {
		logger.Error(err)
		return
	}

	if err := testServerStream(cli); err != nil {
		logger.Error(err)
		return
	}
}

func testUnary(cli *greetconnect.GreetServiceClientImpl) error {
	logger.Info("start to test GRPC_NEW unary call")
	resp, err := cli.Greet(context.Background(), triple.NewRequest(&greet.GreetRequest{Name: "dubbo"}))
	if err != nil {
		return err
	}
	logger.Infof("GRPC_NEW unary call resp: %s", resp.Msg.Greeting)
	return nil
}

func testBidiStream(cli *greetconnect.GreetServiceClientImpl) error {
	logger.Info("start to test GRPC_NEW bidi stream")
	stream, err := cli.GreetStream(context.Background())
	if err != nil {
		return err
	}
	if err := stream.Send(&greet.GreetStreamRequest{Name: "dubbo"}); err != nil {
		return err
	}
	streamResp, err := stream.Receive()
	if err != nil {
		return err
	}
	logger.Infof("GRPC_NEW bidi stream resp: %s", streamResp.Greeting)
	if err := stream.CloseRequest(); err != nil {
		return err
	}
	if err := stream.CloseResponse(); err != nil {
		return err
	}
	return nil
}

func testClientStream(cli *greetconnect.GreetServiceClientImpl) error {
	logger.Info("start to test GRPC_NEW client stream")
	stream, err := cli.GreetClientStream(context.Background())
	if err != nil {
		return err
	}
	for i := 0; i < 5; i++ {
		if err := stream.Send(&greet.GreetClientStreamRequest{Name: "dubbo"}); err != nil {
			return err
		}
	}
	resp, err := stream.CloseAndReceive()
	if err != nil {
		return err
	}
	logger.Infof("GRPC_NEW client stream resp: %s", resp.Msg)
	return nil
}

func testServerStream(cli *greetconnect.GreetServiceClientImpl) error {
	logger.Info("start to test GRPC_NEW server stream")
	stream, err := cli.GreetServerStream(context.Background(), triple.NewRequest(&greet.GreetServerStreamRequest{Name: "dubbo"}))
	if err != nil {
		return err
	}
	for stream.Receive() {
		logger.Infof("GRPC_NEW server stream resp: %s", stream.Msg().Greeting)
	}
	if stream.Err() != nil {
		return err
	}
	if err := stream.Close(); err != nil {
		return err
	}
	return nil
}
