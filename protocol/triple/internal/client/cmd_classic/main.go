package main

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/config"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/triple_gen/greettriple"
	"fmt"
)

import (
	_ "dubbo.apache.org/dubbo-go/v3/imports"
)

func main() {
	svc := new(greettriple.GreetServiceImpl)
	greettriple.SetConsumerService(svc)
	if err := config.Load(); err != nil {
		panic(err)
	}

	resp, err := svc.Greet(context.Background(), &greet.GreetRequest{Name: "dubbo"})
	if err != nil {
		panic(err)
	}

	fmt.Println(resp.Greeting)
}
