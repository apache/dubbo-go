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
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	"github.com/dubbogo/hessian2"
)

import (
	_ "github.com/dubbo/go-for-apache-dubbo/common/proxy/proxy_factory"
	"github.com/dubbo/go-for-apache-dubbo/common/utils"
	"github.com/dubbo/go-for-apache-dubbo/config"
	_ "github.com/dubbo/go-for-apache-dubbo/protocol/dubbo"
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

	hessian.RegisterJavaEnum(Gender(MAN))
	hessian.RegisterJavaEnum(Gender(WOMAN))
	hessian.RegisterPOJO(&User{})

	conMap, _ := config.Load()
	if conMap == nil {
		panic("conMap is nil")
	}

	initProfiling()

	println("\n\n\necho")
	res, err := conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).Echo(context.TODO(), "OK")
	if err != nil {
		panic(err)
	}
	println("res: %v\n", res)

	time.Sleep(3e9)

	println("\n\n\nstart to test dubbo")
	user := &User{}
	err = conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUser(context.TODO(), []interface{}{"A003"}, user)
	if err != nil {
		panic(err)
	}
	println("response result: %v", user)

	println("\n\n\nstart to test dubbo - GetUser0")
	ret, err := conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUser0("A003", "Moorse")
	if err != nil {
		panic(err)
	}
	println("response result: %v", ret)

	println("\n\n\nstart to test dubbo - getUser")
	user = &User{}
	err = conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUser2(context.TODO(), []interface{}{1}, user)
	if err != nil {
		println("getUser - error: %v", err)
	} else {
		println("response result: %v", user)
	}

	println("\n\n\nstart to test dubbo illegal method")
	err = conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUser1(context.TODO(), []interface{}{"A003"}, user)
	if err != nil {
		panic(err)
	}

	initSignal()
}

func initProfiling() {
	if !config.GetConsumerConfig().Pprof_Enabled {
		return
	}
	const (
		PprofPath = "/debug/pprof/"
	)
	var (
		err  error
		ip   string
		addr string
	)

	ip, err = utils.GetLocalIP()
	if err != nil {
		panic("cat not get local ip!")
	}
	addr = ip + ":" + strconv.Itoa(config.GetConsumerConfig().Pprof_Port)
	log.Info("App Profiling startup on address{%v}", addr+PprofPath)

	go func() {
		log.Info(http.ListenAndServe(addr, nil))
	}()
}

func initSignal() {
	signals := make(chan os.Signal, 1)
	// It is not possible to block SIGKILL or syscall.SIGSTOP
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGHUP,
		syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		sig := <-signals
		log.Info("get signal %s", sig.String())
		switch sig {
		case syscall.SIGHUP:
			// reload()
		default:
			go time.AfterFunc(time.Duration(survivalTimeout)*time.Second, func() {
				log.Warn("app exit now by force...")
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
