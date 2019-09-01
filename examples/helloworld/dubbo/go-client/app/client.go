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
	_ "github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config"
	_ "github.com/apache/dubbo-go/protocol/dubbo"
	_ "github.com/apache/dubbo-go/registry/protocol"

	_ "github.com/apache/dubbo-go/filter/impl"

	_ "github.com/apache/dubbo-go/cluster/cluster_impl"
	_ "github.com/apache/dubbo-go/cluster/loadbalance"
	_ "github.com/apache/dubbo-go/registry/zookeeper"
)

var userProvider *UserProvider
var demoProvider *DemoProvider

func init() {
	//userProvider = new(UserProvider)
	//config.SetConsumerService(userProvider)
	//hessian.RegisterPOJO(&User{})

	demoProvider = new(DemoProvider)
	config.SetConsumerService(demoProvider)
}

type User struct {
	Id   string
	Name string
	Age  int32
	Time time.Time
}

type UserProvider struct {
	GetUser func(ctx context.Context, req []interface{}, rsp *User) error
}

func (u *UserProvider) Reference() string {
	return "UserProvider"
}

func (User) JavaClassName() string {
	return "com.ikurento.user.User"
}

type DemoProvider struct {
	SayHello func(ctx context.Context, req []interface{}, rsp string) error
}

func (u *DemoProvider) Reference() string {
	return "DemoProvider"
}

// they are necessary:
// 		export CONF_CONSUMER_FILE_PATH="xxx"
// 		export APP_LOG_CONF_FILE="xxx"
func main() {

	config.Load()
	time.Sleep(3e9)

	println("\n\n\nstart to test dubbo")
	//user := &User{}
	//err := userProvider.GetUser(context.TODO(), []interface{}{"A001"}, user)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Printf("response result: %v\n", user)

	var result string
	err := demoProvider.SayHello(context.Background(), []interface{}{"world"}, result)
	if err != nil {
		panic(err)
	}
	fmt.Printf("demoProvider result: %v\n", result)

}
