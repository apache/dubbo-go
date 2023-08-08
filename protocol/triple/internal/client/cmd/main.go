package main

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/greettriple"
	"github.com/dubbogo/gost/log/logger"
)

func main() {
	//client initialization
	cli, err := client.NewClient(
		client.WithURL("127.0.0.1:8080"),
	)
	dubboCli, err := greettriple.NewGreetServiceClient(cli)
	if err != nil {
		panic(err)
	}

	// classic config initialization
	//dubboCli := new(greettriple.GreetServiceClientImpl)
	//greettriple.SetConsumerService(dubboCli)

	if err := testUnary(dubboCli); err != nil {
		logger.Error(err)
		return
	}

	if err := testBidiStream(dubboCli); err != nil {
		logger.Error(err)
		return
	}

	if err := testClientStream(dubboCli); err != nil {
		logger.Error(err)
		return
	}

	if err := testServerStream(dubboCli); err != nil {
		logger.Error(err)
		return
	}
}

func testUnary(cli greettriple.GreetServiceClient) error {
	logger.Info("start to test TRIPLE unary call")
	resp, err := cli.Greet(context.Background(), &greet.GreetRequest{Name: "triple"})
	if err != nil {
		return err
	}
	logger.Infof("TRIPLE unary call resp: %s", resp.Greeting)
	return nil
}

func testBidiStream(cli greettriple.GreetServiceClient) error {
	logger.Info("start to test TRIPLE bidi stream")
	stream, err := cli.GreetStream(context.Background())
	if err != nil {
		return err
	}
	if err := stream.Send(&greet.GreetStreamRequest{Name: "triple"}); err != nil {
		return err
	}
	resp, err := stream.Recv()
	if err != nil {
		return err
	}
	logger.Infof("TRIPLE bidi stream resp: %s", resp.Greeting)
	if err := stream.CloseRequest(); err != nil {
		return err
	}
	if err := stream.CloseResponse(); err != nil {
		return err
	}
	return nil
}

func testClientStream(cli greettriple.GreetServiceClient) error {
	logger.Info("start to test TRIPLE dubboClint stream")
	stream, err := cli.GreetClientStream(context.Background())
	if err != nil {
		return err
	}
	for i := 0; i < 5; i++ {
		if err := stream.Send(&greet.GreetClientStreamRequest{Name: "triple"}); err != nil {
			return err
		}
	}
	resp, err := stream.CloseAndRecv()
	if err != nil {
		return err
	}
	logger.Infof("TRIPLE dubboClint stream resp: %s", resp.Greeting)
	return nil
}

func testServerStream(cli greettriple.GreetServiceClient) error {
	logger.Info("start to test TRIPLE server stream")
	stream, err := cli.GreetServerStream(context.Background(), &greet.GreetServerStreamRequest{Name: "triple"})
	if err != nil {
		return err
	}
	for stream.Recv() {
		logger.Infof("TRIPLE server stream resp: %s", stream.Msg().Greeting)
	}
	if stream.Err() != nil {
		return err
	}
	if err := stream.Close(); err != nil {
		return err
	}
	return nil
}
