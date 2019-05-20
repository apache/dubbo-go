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
	"github.com/AlexStocks/goext/log"
	"github.com/AlexStocks/goext/net"
	log "github.com/AlexStocks/log4go"
	"github.com/dubbogo/hessian2"
)

import (
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

	gxlog.CInfo("\n\n\necho")
	res := ""
	err := conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).Echo(context.TODO(), []interface{}{"OK"}, &res)
	if err != nil {
		panic(err)
	}
	gxlog.CInfo("res: %s", res)

	time.Sleep(3e9)

	gxlog.CInfo("\n\n\nstart to test dubbo")
	user := &User{}
	err = conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUser(context.TODO(), []interface{}{"A003"}, user)
	if err != nil {
		panic(err)
	}
	gxlog.CInfo("response result: %v", user)

	gxlog.CInfo("\n\n\nstart to test dubbo - getUser")
	user = &User{}
	err = conMap["com.ikurento.user.UserProvider"].GetRPCService().(*UserProvider).GetUser2(context.TODO(), []interface{}{1}, user)
	if err != nil {
		fmt.Println("getUser - error: ", err)
	}
	gxlog.CInfo("response result: %v", user)

	gxlog.CInfo("\n\n\nstart to test dubbo illegal method")
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

	ip, err = gxnet.GetLocalIP()
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
