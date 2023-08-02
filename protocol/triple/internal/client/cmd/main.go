package main

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/consumer"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/greettriple"
	tri "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
	"github.com/dubbogo/gost/log/logger"
)

func main() {
	pro, err := provider.NewProvider(opt...)
	greetriple.Register(pro, service)
}

func main() {
	//root, err := dubbo.NewRootWithYaml("")
	////root, err := dubbo.NewRoot(
	////	dubboConsumer.WithURL("127.0.0.1:8080"),
	////	)
	con, err := consumer.NewConsumer(
		consumer.WithURL("127.0.0.1:8080"),
		consumer.WithRegitry(WithNacos())
		c
	)
	consumer -> registry -> remoting
	if err != nil {
		panic(err)
	}
	dubboIns, err := dubbo.NewInstance(opts...)
	cli, err := greettriple.NewGreetServiceClient(dubboIns)
	if err != nil {
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

func testUnary(cli greettriple.GreetServiceClient) error {
	logger.Info("start to test GRPC_NEW unary call")
	resp, err := cli.Greet(context.Background(), &greet.GreetRequest{Name: "dubbo"})
	if err != nil {
		return err
	}
	logger.Infof("GRPC_NEW unary call resp: %s", resp.Greeting)
	return nil
}

func testBidiStream(cli greettriple.GreetServiceClient) error {
	logger.Info("start to test GRPC_NEW bidi stream")
	stream, err := cli.GreetStream(context.Background())
	if err != nil {
		return err
	}
	if err := stream.Send(&greet.GreetStreamRequest{Name: "dubbo"}); err != nil {
		return err
	}
	streamResp := new(greet.GreetStreamResponse)
	if err := stream.Receive(streamResp); err != nil {
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

func testClientStream(cli greettriple.GreetServiceClient) error {
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
	respRaw := new(greet.GreetClientStreamResponse)
	resp := tri.NewResponse(respRaw)
	if err := stream.CloseAndReceive(resp); err != nil {
		return err
	}
	logger.Infof("GRPC_NEW client stream resp: %s", respRaw.Greeting)
	return nil
}

func testServerStream(cli greettriple.GreetServiceClient) error {
	logger.Info("start to test GRPC_NEW server stream")
	stream, err := cli.GreetServerStream(context.Background(), &greet.GreetServerStreamRequest{Name: "dubbo"})
	if err != nil {
		return err
	}
	msg := new(greet.GreetServerStreamResponse)
	for stream.Receive(msg) {
		logger.Infof("GRPC_NEW server stream resp: %s", msg.Greeting)
	}
	if stream.Err() != nil {
		return err
	}
	if err := stream.Close(); err != nil {
		return err
	}
	return nil
}
