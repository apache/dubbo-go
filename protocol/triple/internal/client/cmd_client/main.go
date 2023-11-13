/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package main

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/client/common"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/triple_gen/greettriple"
	"fmt"
	"time"
)

func main() {
	ins, err := dubbo.NewInstance(
		dubbo.WithName("dubbo_test"),
	)

	// for the most brief RPC case
	cli, err := ins.NewClient(
		client.WithClientURL("127.0.0.1:20000"),
	)

	if err != nil {
		panic(err)
	}
	svc, err := greettriple.NewGreetService(cli)

	if err != nil {
		panic(err)
	}

	response, err := svc.Greet(context.Background(), &greet.GreetRequest{Name: "greet"}, client.WithCallRequestTimeout(5*time.Second))
	if err != nil {
		panic(err)
	}

	fmt.Printf("result: %s", response.Greeting)

	common.TestClient(svc)
}
