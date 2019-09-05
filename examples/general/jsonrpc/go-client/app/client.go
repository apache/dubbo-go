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
	"os"
	"os/signal"
	"syscall"
	"time"
)

import (
	"github.com/apache/dubbo-go/common/logger"
	_ "github.com/apache/dubbo-go/common/proxy/proxy_factory"
	"github.com/apache/dubbo-go/config"
	_ "github.com/apache/dubbo-go/protocol/jsonrpc"
	_ "github.com/apache/dubbo-go/registry/protocol"

	_ "github.com/apache/dubbo-go/filter/impl"

	_ "github.com/apache/dubbo-go/cluster/cluster_impl"
	_ "github.com/apache/dubbo-go/cluster/loadbalance"
	_ "github.com/apache/dubbo-go/registry/zookeeper"
)

var (
	survivalTimeout int = 10e9
)

// they are necessary:
// 		export CONF_CONSUMER_FILE_PATH="xxx"
// 		export APP_LOG_CONF_FILE="xxx"
func main() {

	config.Load()

	println("\n\ntest")
	test()
	println("\n\ntest1")
	test1()
	println("\n\ntest2")
	test2()

	initSignal()
}

func initSignal() {
	signals := make(chan os.Signal, 1)
	// It is not possible to block SIGKILL or syscall.SIGSTOP
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGHUP,
		syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		sig := <-signals
		logger.Infof("get signal %s", sig.String())
		switch sig {
		case syscall.SIGHUP:
		// reload()
		default:
			time.AfterFunc(time.Duration(survivalTimeout), func() {
				logger.Warnf("app exit now by force...")
				os.Exit(1)
			})

			// The program exits normally or timeout forcibly exits.
			fmt.Println("app exit now...")
			return
		}
	}
}

func println(format string, args ...interface{}) {
	fmt.Printf("\033[32;40m"+format+"\033[0m\n", args...)
}

func test() {
	println("\n\n\necho")
	res, err := userProvider.Echo(context.TODO(), "OK")
	if err != nil {
		println("echo - error: %v", err)
	} else {
		println("res: %v", res)
	}

	time.Sleep(10e9)

	println("\n\n\nstart to test jsonrpc")
	user := &JsonRPCUser{}
	err = userProvider.GetUser(context.TODO(), []interface{}{"A003"}, user)
	if err != nil {
		panic(err)
	}
	println("response result: %v", user)

	println("\n\n\nstart to test jsonrpc - GetUser0")
	ret, err := userProvider.GetUser0("A003", "Moorse")
	if err != nil {
		panic(err)
	}
	println("response result: %v", ret)

	println("\n\n\nstart to test jsonrpc - GetUsers")
	ret1, err := userProvider.GetUsers([]interface{}{[]interface{}{"A002", "A003"}})
	if err != nil {
		panic(err)
	}
	println("response result: %v", ret1)

	println("\n\n\nstart to test jsonrpc - getUser")
	user = &JsonRPCUser{}
	err = userProvider.GetUser2(context.TODO(), []interface{}{1}, user)
	if err != nil {
		panic(err)
	}
	println("response result: %v", user)

	println("\n\n\nstart to test jsonrpc - GetUser3")
	err = userProvider.GetUser3()
	if err != nil {
		panic(err)
	}
	println("succ!")

	println("\n\n\nstart to test jsonrpc illegal method")
	err = userProvider.GetUser1(context.TODO(), []interface{}{"A003"}, user)
	if err == nil {
		panic("err is nil")
	}
	println("error: %v", err)
}

func test1() {
	println("\n\n\necho")
	res, err := userProvider1.Echo(context.TODO(), "OK")
	if err != nil {
		println("echo - error: %v", err)
	} else {
		println("res: %v", res)
	}

	time.Sleep(3e9)

	println("\n\n\nstart to test jsonrpc")
	user := &JsonRPCUser{}
	err = userProvider1.GetUser(context.TODO(), []interface{}{"A003"}, user)
	if err != nil {
		panic(err)
	}
	println("response result: %v", user)

	println("\n\n\nstart to test jsonrpc - GetUser0")
	ret, err := userProvider1.GetUser0("A003", "Moorse")
	if err != nil {
		panic(err)
	}
	println("response result: %v", ret)

	println("\n\n\nstart to test jsonrpc - GetUsers")
	ret1, err := userProvider1.GetUsers([]interface{}{[]interface{}{"A002", "A003"}})
	if err != nil {
		panic(err)
	}
	println("response result: %v", ret1)

	println("\n\n\nstart to test jsonrpc - getUser")
	user = &JsonRPCUser{}
	err = userProvider1.GetUser2(context.TODO(), []interface{}{1}, user)
	if err != nil {
		panic(err)
	}
	println("response result: %v", user)

	println("\n\n\nstart to test jsonrpc - GetUser3")
	err = userProvider1.GetUser3()
	if err != nil {
		panic(err)
	}
	println("succ!")

	println("\n\n\nstart to test jsonrpc illegal method")
	err = userProvider1.GetUser1(context.TODO(), []interface{}{"A003"}, user)
	if err == nil {
		panic("err is nil")
	}
	println("error: %v", err)
}

func test2() {
	println("\n\n\necho")
	res, err := userProvider2.Echo(context.TODO(), "OK")
	if err != nil {
		println("echo - error: %v", err)
	} else {
		println("res: %v", res)
	}

	time.Sleep(3e9)

	println("\n\n\nstart to test jsonrpc")
	user := &JsonRPCUser{}
	err = userProvider2.GetUser(context.TODO(), []interface{}{"A003"}, user)
	if err != nil {
		panic(err)
	}
	println("response result: %v", user)

	println("\n\n\nstart to test jsonrpc - GetUser0")
	ret, err := userProvider2.GetUser0("A003", "Moorse")
	if err != nil {
		panic(err)
	}
	println("response result: %v", ret)

	println("\n\n\nstart to test jsonrpc - GetUsers")
	ret1, err := userProvider2.GetUsers([]interface{}{[]interface{}{"A002", "A003"}})
	if err != nil {
		panic(err)
	}
	println("response result: %v", ret1)

	println("\n\n\nstart to test jsonrpc - getUser")
	user = &JsonRPCUser{}
	err = userProvider2.GetUser2(context.TODO(), []interface{}{1}, user)
	if err != nil {
		panic(err)
	}
	println("response result: %v", user)

	println("\n\n\nstart to test jsonrpc - GetUser3")
	err = userProvider2.GetUser3()
	if err != nil {
		panic(err)
	}
	println("succ!")

	println("\n\n\nstart to test jsonrpc illegal method")
	err = userProvider2.GetUser1(context.TODO(), []interface{}{"A003"}, user)
	if err == nil {
		panic("err is nil")
	}
	println("error: %v", err)
}
