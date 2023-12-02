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
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/client"
	_ "dubbo.apache.org/dubbo-go/v3/imports"
)

func main() {
	cli, err := client.NewClient(
		client.WithClientProtocolDubbo(),
	)
	if err != nil {
		panic(err)
	}
	conn, err := cli.Dial("GreetProvider",
		client.WithURL("127.0.0.1:20000"),
	)
	if err != nil {
		panic(err)
	}
	var resp string
	if err := conn.CallUnary(context.Background(), []interface{}{"hello", "new", "dubbo"}, &resp, "Greet"); err != nil {
		logger.Errorf("GreetProvider.Greet err: %s", err)
		return
	}
	logger.Infof("Get Response: %s", resp)
}
