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
	"fmt"
	"time"
)

import (
	hessian "github.com/apache/dubbo-go-hessian2"
	_ "github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config"
	_ "github.com/apache/dubbo-go/protocol/dubbo"
	_ "github.com/apache/dubbo-go/registry/protocol"

	_ "github.com/apache/dubbo-go/filter/filter_impl"

	_ "github.com/apache/dubbo-go/cluster/cluster_impl"
	_ "github.com/apache/dubbo-go/cluster/loadbalance"
	_ "github.com/apache/dubbo-go/registry/zookeeper"
)

var (
	survivalTimeout int = 10e9
)

func println(format string, args ...interface{}) {
	fmt.Printf("\033[32;40m"+format+"\033[0m\n", args...)
}

// they are necessary:
// 		export CONF_CONSUMER_FILE_PATH="xxx"
// 		export APP_LOG_CONF_FILE="xxx"
func main() {
	hessian.RegisterPOJO(&User{})
	config.Load()
	time.Sleep(3e9)

	println("\n\n\nstart to test dubbo")

	go func() {
		select {
		case <-time.After(time.Minute):
			panic("provider not start after client already running 1min")
		}
	}()
	user := &User{}
	err := userProvider.GetUser(context.TODO(), []interface{}{"A001"}, user)
	if err != nil {
		panic(err)
	}
	println("response result: %v\n", user)
}
