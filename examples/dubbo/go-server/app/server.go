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
	"github.com/dubbogo/hessian2"
	log "github.com/dubbogo/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/dubbo"
	"github.com/dubbo/dubbo-go/plugins"
	"github.com/dubbo/dubbo-go/registry"
	"github.com/dubbo/dubbo-go/registry/zookeeper"
)

var (
	survivalTimeout = int(3e9)
	servo           *dubbo.Server
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

	hessian.RegisterJavaEnum(Gender(MAN))
	hessian.RegisterJavaEnum(Gender(WOMAN))
	hessian.RegisterPOJO(&DubboUser{})

	servo = initServer()
	err = servo.Register(&UserProvider{})
	if err != nil {
		panic(err)
		return
	}
	servo.Start()

	initSignal()
}

func initServer() *dubbo.Server {
	var (
		srv *dubbo.Server
	)

	if conf == nil {
		panic(fmt.Sprintf("conf is nil"))
		return nil
	}

	// registry

	regs, err := plugins.PluggableRegistries[conf.Registry](
		registry.WithDubboType(registry.PROVIDER),
		registry.WithApplicationConf(conf.Application_Config),
		zookeeper.WithRegistryConf(conf.ZkRegistryConfig),
	)

	if err != nil || regs == nil {
		panic(fmt.Sprintf("fail to init registry.Registy, err:%s", jerrors.ErrorStack(err)))
		return nil
	}

	// generate server config
	serverConfig := make([]dubbo.ServerConfig, len(conf.Server_List))
	for i := 0; i < len(conf.Server_List); i++ {
		serverConfig[i] = dubbo.ServerConfig{
			SessionNumber:   700,
			FailFastTimeout: "5s",
			SessionTimeout:  "20s",
			GettySessionParam: dubbo.GettySessionParam{
				CompressEncoding: false, // 必须false
				TcpNoDelay:       true,
				KeepAlivePeriod:  "120s",
				TcpRBufSize:      262144,
				TcpKeepAlive:     true,
				TcpWBufSize:      65536,
				PkgRQSize:        1024,
				PkgWQSize:        512,
				TcpReadTimeout:   "1s",
				TcpWriteTimeout:  "5s",
				WaitTimeout:      "1s",
				MaxMsgLen:        1024,
				SessionName:      "server",
			},
		}
		serverConfig[i].IP = conf.Server_List[i].IP
		serverConfig[i].Port = conf.Server_List[i].Port
		serverConfig[i].Protocol = conf.Server_List[i].Protocol
	}

	// provider
	srv = dubbo.NewServer(
		dubbo.Registry(regs),
		dubbo.ConfList(serverConfig),
		dubbo.ServiceConfList(conf.ServiceConfigList),
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
