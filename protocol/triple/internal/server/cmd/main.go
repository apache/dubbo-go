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
	"dubbo.apache.org/dubbo-go/v3"
	"dubbo.apache.org/dubbo-go/v3/global"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/proto/triple_gen/greettriple"
	"dubbo.apache.org/dubbo-go/v3/protocol/triple/internal/server/api"
	"dubbo.apache.org/dubbo-go/v3/server"
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
	srv, err := ins.NewServer(
		server.WithServer_ProtocolConfig("tri",
			global.WithProtocol_Port("20000"),
			global.WithProtocol_Name("tri"),
		))
	if err != nil {
		panic(err)
	}
	if err := greettriple.RegisterGreetServiceHandler(srv, &api.GreetTripleServer{},
		server.WithProtocolIDs([]string{"tri"}),
		server.WithNotRegister(true),
	); err != nil {
		panic(err)
	}
	select {}
}
