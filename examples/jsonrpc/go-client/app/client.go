// Copyright 2016-2019 Yincheng Fang, Alex Stocks
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
	"github.com/dubbo/go-for-apache-dubbo/common/logger"
	_ "github.com/dubbo/go-for-apache-dubbo/common/proxy/proxy_factory"
	"github.com/dubbo/go-for-apache-dubbo/config"
	_ "github.com/dubbo/go-for-apache-dubbo/protocol/jsonrpc"
	_ "github.com/dubbo/go-for-apache-dubbo/registry/protocol"

	_ "github.com/dubbo/go-for-apache-dubbo/filter/impl"

	_ "github.com/dubbo/go-for-apache-dubbo/cluster/cluster_impl"
	_ "github.com/dubbo/go-for-apache-dubbo/cluster/loadbalance"
	_ "github.com/dubbo/go-for-apache-dubbo/registry/zookeeper"
)

var (
	survivalTimeout int = 10e9
)

// they are necessary:
// 		export CONF_CONSUMER_FILE_PATH="xxx"
// 		export APP_LOG_CONF_FILE="xxx"
func main() {

	conMap, _ := config.Load()
	if conMap == nil {
		panic("conMap is nil")
	}

	println("\n\n\necho")
	res, err := conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).Echo(context.TODO(), "OK")
	if err != nil {
		println("echo - error: %v", err)
	} else {
		println("res: %v", res)
	}

	time.Sleep(3e9)

	println("\n\n\nstart to test jsonrpc")
	user := &JsonRPCUser{}
	err = conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUser(context.TODO(), []interface{}{"A003"}, user)
	if err != nil {
		panic(err)
	}
	println("response result: %v", user)

	println("\n\n\nstart to test jsonrpc - GetUser0")
	ret, err := conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUser0("A003", "Moorse")
	if err != nil {
		panic(err)
	}
	println("response result: %v", ret)

	println("\n\n\nstart to test jsonrpc - GetUsers")
	ret1, err := conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUsers([]interface{}{[]interface{}{"A002", "A003"}})
	if err != nil {
		panic(err)
	}
	println("response result: %v", ret1)

	println("\n\n\nstart to test jsonrpc - getUser")
	user = &JsonRPCUser{}
	err = conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUser2(context.TODO(), []interface{}{1}, user)
	if err != nil {
		println("getUser - error: %v", err)
	} else {
		println("response result: %v", user)
	}

	println("\n\n\nstart to test jsonrpc illegal method")
	err = conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUser1(context.TODO(), []interface{}{"A003"}, user)
	if err != nil {
		panic(err)
	}

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
			go time.AfterFunc(time.Duration(survivalTimeout)*time.Second, func() {
				logger.Warnf("app exit now by force...")
				os.Exit(1)
			})

			// 要么fastFailTimeout时间内执行完毕下面的逻辑然后程序退出，要么执行上面的超时函数程序强行退出
			fmt.Println("app exit now...")
			return
		}
	}
}

func println(format string, args ...interface{}) {
	fmt.Printf("\033[32;40m"+format+"\033[0m\n", args...)
}
