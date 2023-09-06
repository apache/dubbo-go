package main

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/client"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/triple_gen/greettriple"
	"fmt"
)

func main() {
	// for the most brief RPC case
	cli, err := client.NewClient(
		client.WithURL("127.0.0.1:20000"),
	)
	if err != nil {
		panic(err)
	}
	svc, err := greettriple.NewGreetService(cli)
	if err != nil {
		panic(err)
	}

	resp, err := svc.Greet(context.Background(), &greet.GreetRequest{Name: "dubbo"})
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Greeting)
}
