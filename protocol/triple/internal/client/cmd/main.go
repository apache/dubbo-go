package main

import (
	"context"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/greettriple"
)

func main() {
	//client initialization
	cli, err := client.NewClient(
		client.WithURL("127.0.0.1:8080"),
	)
	svc, err := greettriple.NewGreetService(cli)
	if err != nil {
		panic(err)
	}

	// classic config initialization
	//svc := new(greettriple.GreetServiceImpl)
	//greettriple.SetConsumerService(dubboCli)
	//config.Load() // need dubbogo.yml

	if err := testUnary(svc); err != nil {
		logger.Error(err)
		return
	}

	if err := testBidiStream(svc); err != nil {
		logger.Error(err)
		return
	}

	if err := testClientStream(svc); err != nil {
		logger.Error(err)
		return
	}

	if err := testServerStream(svc); err != nil {
		logger.Error(err)
		return
	}
}

func testUnary(svc greettriple.GreetService) error {
	logger.Info("start to test TRIPLE unary call")
	resp, err := svc.Greet(context.Background(), &greet.GreetRequest{Name: "triple"})
	if err != nil {
		return err
	}
	logger.Infof("TRIPLE unary call resp: %s", resp.Greeting)
	return nil
}

func testBidiStream(svc greettriple.GreetService) error {
	logger.Info("start to test TRIPLE bidi stream")
	stream, err := svc.GreetStream(context.Background())
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

func testClientStream(svc greettriple.GreetService) error {
	logger.Info("start to test TRIPLE dubboClint stream")
	stream, err := svc.GreetClientStream(context.Background())
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

func testServerStream(svc greettriple.GreetService) error {
	logger.Info("start to test TRIPLE server stream")
	stream, err := svc.GreetServerStream(context.Background(), &greet.GreetServerStreamRequest{Name: "triple"})
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
