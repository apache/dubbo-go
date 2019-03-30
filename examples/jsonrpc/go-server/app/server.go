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
	// "github.com/AlexStocks/gocolor"
	"github.com/AlexStocks/goext/net"
	"github.com/AlexStocks/goext/time"
	log "github.com/AlexStocks/log4go"
	"github.com/dubbo/dubbo-go/jsonrpc"
	"github.com/dubbo/dubbo-go/registry"
	jerrors "github.com/juju/errors"
)

var (
	survivalTimeout = int(3e9)
	servo           *jsonrpc.Server
)

func main() {
	var (
		err error
	)

	err = configInit()
	if err != nil {
		log.Error("configInit() = error{%#v}", err)
		return
	}
	initProfiling()

	servo = initServer()
	err = servo.Handle(&UserProvider{})
	if err != nil {
		panic(err)
		return
	}
	servo.Start()

	initSignal()
}

func initServer() *jsonrpc.Server {
	var (
		err            error
		serverRegistry *registry.ZkProviderRegistry
		srv            *jsonrpc.Server
	)

	if conf == nil {
		panic(fmt.Sprintf("conf is nil"))
		return nil
	}

	// registry
	serverRegistry, err = registry.NewZkProviderRegistry(
		registry.ApplicationConf(conf.Application_Config),
		registry.RegistryConf(conf.Registry_Config),
		registry.BalanceMode(registry.SM_RoundRobin),
		registry.ServiceTTL(conf.netTimeout),
	)
	if err != nil || serverRegistry == nil {
		panic(fmt.Sprintf("fail to init registry.Registy, err:%s", jerrors.ErrorStack(err)))
		return nil
	}

	// provider
	srv = jsonrpc.NewServer(
		jsonrpc.Registry(serverRegistry),
		jsonrpc.ConfList(conf.Server_List),
		jsonrpc.ServiceConfList(conf.Service_List),
	)

	return srv
}

func uninitServer() {
	if servo != nil {
		servo.Stop()
	}
	log.Close()
}

func initProfiling() {
	if !conf.Pprof_Enabled {
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
	addr = ip + ":" + strconv.Itoa(conf.Pprof_Port)
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
			uninitServer()
			fmt.Println("provider app exit now...")
			return
		}
	}
}
