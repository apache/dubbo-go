// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This client is intentionally pinned to dubbo-go v3.3.2. It exercises the
// legacy Triple invoker against the current checkout's Go Triple server.
package main

import (
	"context"
	"fmt"
	"reflect"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple"
)

type greetRequest struct {
	Name string
}

func (*greetRequest) JavaClassName() string {
	return "org.apache.dubbo.tri.hessian2.api.GreetRequest"
}

type greetResponse struct {
	Greeting string
}

func (*greetResponse) JavaClassName() string {
	return "org.apache.dubbo.tri.hessian2.api.GreetResponse"
}

func main() {
	hessian.RegisterPOJO(new(greetRequest))
	hessian.RegisterPOJO(new(greetResponse))

	url, err := common.NewURL("tri://127.0.0.1:20000/org.apache.dubbo.tri.hessian2.api.GreetingsService",
		common.WithParamsValue("interface", "org.apache.dubbo.tri.hessian2.api.GreetingsService"),
		common.WithParamsValue("serialization", "hessian2"),
		common.WithParamsValue("timeout", "5s"),
	)
	if err != nil {
		panic(err)
	}
	client, err := triple.NewDubbo3Invoker(url)
	if err != nil {
		panic(err)
	}
	defer client.Destroy()

	response := new(greetResponse)
	invocation := invocation.NewRPCInvocationWithOptions(
		invocation.WithMethodName("Greet"),
		invocation.WithParameterValues([]reflect.Value{reflect.ValueOf(&greetRequest{Name: "legacy-dubbo-go"})}),
		invocation.WithReply(response),
	)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := client.Invoke(ctx, invocation)
	if err := result.Error(); err != nil {
		panic(err)
	}
	if response.Greeting != "The name of the request is:\t legacy-dubbo-go" {
		panic(fmt.Sprintf("unexpected response: %q", response.Greeting))
	}
	fmt.Println(response.Greeting)
}
