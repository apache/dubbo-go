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
	"fmt"
	"time"
)

import (
	_ "github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol/dubbo"
	_ "github.com/apache/dubbo-go/registry/protocol"

	_ "github.com/apache/dubbo-go/filter/impl"

	_ "github.com/apache/dubbo-go/cluster/cluster_impl"
	_ "github.com/apache/dubbo-go/cluster/loadbalance"
	_ "github.com/apache/dubbo-go/registry/zookeeper"
)

// they are necessary:
// 		export CONF_CONSUMER_FILE_PATH="xxx"
// 		export APP_LOG_CONF_FILE="xxx"
func main() {
	println("\n\ntest")
	test()
	println("\n\ntest2")
	test2()

}
func test() {
	var appName = "UserProviderGer"
	var referenceConfig = config.ReferenceConfig{
		InterfaceName: "com.ikurento.user.UserProvider",
		Cluster:       "failover",
		Registry:      "hangzhouzk",
		Protocol:      dubbo.DUBBO,
		Generic:       true,
	}
	referenceConfig.GenericLoad(appName) //appName is the unique identification of RPCService

	time.Sleep(3 * time.Second)
	println("\n\n\nstart to generic invoke")
	resp, err := referenceConfig.GetRPCService().(*config.GenericService).Invoke([]interface{}{"GetUser", []string{"java.lang.String"}, []interface{}{"A003"}})
	if err != nil {
		panic(err)
	}
	fmt.Printf("res: %+v\n", resp)
	println("succ!")

}
func test2() {
	var appName = "UserProviderGer"
	var referenceConfig = config.ReferenceConfig{
		InterfaceName: "com.ikurento.user.UserProvider",
		Cluster:       "failover",
		Registry:      "hangzhouzk",
		Protocol:      dubbo.DUBBO,
		Generic:       true,
	}
	referenceConfig.GenericLoad(appName) //appName is the unique identification of RPCService

	time.Sleep(3 * time.Second)
	println("\n\n\nstart to generic invoke")
	user := User{
		Id:   "3213",
		Name: "panty",
		Age:  25,
		Time: time.Now(),
		Sex:  Gender(MAN),
	}
	resp, err := referenceConfig.GetRPCService().(*config.GenericService).Invoke([]interface{}{"queryUser", []string{"com.ikurento.user.User"}, []interface{}{user}})
	if err != nil {
		panic(err)
	}
	fmt.Printf("res: %+v\n", resp)
	println("succ!")

}
