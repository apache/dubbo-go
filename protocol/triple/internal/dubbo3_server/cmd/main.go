package main

import (
	"dubbo.apache.org/dubbo-go/v3/config"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/dubbo3_server/api"
)

func main() {
	config.SetProviderService(&api.GreetDubbo3Server{})
	if err := config.Load(config.WithPath("./dubbogo.yml")); err != nil {
		panic(err)
	}
	select {}
}
