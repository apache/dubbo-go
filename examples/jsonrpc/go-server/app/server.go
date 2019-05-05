package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

import (
	"github.com/AlexStocks/goext/net"
	"github.com/AlexStocks/goext/time"
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/dubbo-go/config/support"

	_ "github.com/dubbo/dubbo-go/protocol/dubbo"
	_ "github.com/dubbo/dubbo-go/protocol/jsonrpc"
	_ "github.com/dubbo/dubbo-go/registry/protocol"

	_ "github.com/dubbo/dubbo-go/filter/imp"

	_ "github.com/dubbo/dubbo-go/cluster/loadbalance"
	_ "github.com/dubbo/dubbo-go/cluster/support"
	_ "github.com/dubbo/dubbo-go/registry/zookeeper"
)

var (
	survivalTimeout = int(3e9)
)

// they are necessary:
// 		export CONF_CONSUMER_FILE_PATH="xxx"
// 		export CONF_PROVIDER_FILE_PATH="xxx"
// 		export APP_LOG_CONF_FILE="xxx"
func main() {

	_, proMap := support.Load()
	if proMap == nil {
		panic("proMap is nil")
	}

	initProfiling()

	//todo

	initSignal()
}

func initProfiling() {
	if !support.GetProviderConfig().Pprof_Enabled {
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
	addr = ip + ":" + strconv.Itoa(support.GetProviderConfig().Pprof_Port)
	log.Info("App Profiling startup on address{%v}", addr+PprofPath)

	go func() {
		log.Info(http.ListenAndServe(addr, nil))
	}()
}

func initSignal() {
	signals := make(chan os.Signal, 1)
	// It is not possible to block SIGKILL or syscall.SIGSTOP
	signal.Notify(signals, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		sig := <-signals
		log.Info("get signal %s", sig.String())
		switch sig {
		case syscall.SIGHUP:
		// reload()
		default:
			go gxtime.Future(survivalTimeout, func() {
				log.Warn("app exit now by force...")
				os.Exit(1)
			})

			// 要么fastFailTimeout时间内执行完毕下面的逻辑然后程序退出，要么执行上面的超时函数程序强行退出
			// todo
			fmt.Println("provider app exit now...")
			return
		}
	}
}
