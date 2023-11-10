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
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/client/common"
	greet "dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/triple_gen/greettriple"
	"time"
)

func main() {
	// for the most brief RPC case
	cli, err := client.NewClient(
		client.WithClientURL("127.0.0.1:20000"),
		client.WithClientProtocolDubbo(),
	)
	if err != nil {
		panic(err)
	}
	svc, err := greettriple.NewGreetService(cli, client.WithCheck(), client.WithProtocolDubbo())

	svc2, err := greettriple.NewGreetService(cli, client.WithCheck(), client.WithProtocolDubbo())
	if err != nil {
		panic(err)
	}

	svc.Greet(context.Background(), &greet.GreetRequest{Name: "greet"}, client.WithCallRequestTimeout(5*time.Second))
	svc2.Greet(context.Background(), &greet.GreetRequest{Name: "greet"}, client.WithCallRequestTimeout(5*time.Second))

	common.TestClient(svc)
}
