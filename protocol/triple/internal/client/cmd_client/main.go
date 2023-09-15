package main

import (
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/client/common"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/triple_gen/greettriple"
)

func main() {
	// for the most brief RPC case
	cli, err := client.NewClient(
		client.WithProtocol("tri"),
		client.WithURL("tri://127.0.0.1:20000"),
	)
	if err != nil {
		panic(err)
	}
	svc, err := greettriple.NewGreetService(cli)
	if err != nil {
		panic(err)
	}

	common.TestClient(svc)
}
