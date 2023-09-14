package main

import (
	"context"
	"fmt"
)

import (
	dubbo "dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/client"
	"dubbo.apache.org/dubbo-go/v3/global"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/triple_gen/greettriple"
)

func main() {
	// global conception
	// configure global configurations and common modules
	ins, err := dubbo.NewInstance(
		dubbo.WithApplication(
			global.WithApplication_Name("dubbo_test"),
		),
		//dubbo.WithRegistry("nacos",
		//	global.WithRegistry_Address("127.0.0.1:8848"),
		//),
		dubbo.WithMetric(
			global.WithMetric_Enable(true),
		),
	)
	if err != nil {
		panic(err)
	}
	// configure the params that only client layer cares
	cli, err := ins.NewClient(
		client.WithProtocol("triple"),
		client.WithRetries(3),
		client.WithURL("triple://127.0.0.1:20000"),
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
