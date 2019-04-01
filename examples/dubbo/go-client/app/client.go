package main

import (
	"fmt"
	"github.com/dubbo/dubbo-go/dubbo"
	"github.com/dubbo/dubbo-go/plugins"
	"github.com/dubbo/dubbo-go/registry/zookeeper"
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
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/client/invoker"
	"github.com/dubbo/dubbo-go/examples"
	"github.com/dubbo/dubbo-go/public"
	"github.com/dubbo/dubbo-go/registry"
)

var (
	survivalTimeout int = 10e9
	clientInvoker   *invoker.Invoker
)

func main() {

	clientConfig := examples.InitClientConfig()
	initProfiling(clientConfig)
	initClient(clientConfig)

	time.Sleep(3e9)

	gxlog.CInfo("\n\n\nstart to test dubbo")
	testDubborpc(clientConfig, "A003")

	time.Sleep(3e9)

	initSignal()
}

func initClient(clientConfig *examples.ClientConfig) {
	var (
		err       error
		codecType public.CodecType
	)

	if clientConfig == nil {
		panic(fmt.Sprintf("clientConfig is nil"))
		return
	}

	// registry
	clientRegistry, err := plugins.PluggableRegistries[clientConfig.Registry](
		registry.WithDubboType(registry.CONSUMER),
		registry.WithApplicationConf(clientConfig.Application_Config),
		zookeeper.WithRegistryConf(clientConfig.ZkRegistryConfig),
	)
	if err != nil {
		panic(fmt.Sprintf("fail to init registry.Registy, err:%s", jerrors.ErrorStack(err)))
		return
	}

	// consumer
	clientConfig.RequestTimeout, err = time.ParseDuration(clientConfig.Request_Timeout)
	if err != nil {
		panic(fmt.Sprintf("time.ParseDuration(Request_Timeout{%#v}) = error{%v}",
			clientConfig.Request_Timeout, err))
		return
	}
	clientConfig.ConnectTimeout, err = time.ParseDuration(clientConfig.Connect_Timeout)
	if err != nil {
		panic(fmt.Sprintf("time.ParseDuration(Connect_Timeout{%#v}) = error{%v}",
			clientConfig.Connect_Timeout, err))
		return
	}

	for idx := range clientConfig.Service_List {
		codecType = public.GetCodecType(clientConfig.Service_List[idx].Protocol)
		if codecType == public.CODECTYPE_UNKNOWN {
			panic(fmt.Sprintf("unknown protocol %s", clientConfig.Service_List[idx].Protocol))
		}
	}

	for _, service := range clientConfig.Service_List {
		err = clientRegistry.RegisterConsumer(service)
		if err != nil {
			panic(fmt.Sprintf("registry.Register(service{%#v}) = error{%v}", service, jerrors.ErrorStack(err)))
			return
		}
	}

	//read the client lb config in config.yml
	configClientLB := plugins.PluggableLoadbalance[clientConfig.ClientLoadBalance]()

	//init dubbo rpc client & init invoker
	var cltD *dubbo.Client

	cltD, err = dubbo.NewClient(&dubbo.ClientConfig{
		PoolSize:        64,
		PoolTTL:         600,
		ConnectionNum:   2, // 不能太大
		FailFastTimeout: "5s",
		SessionTimeout:  "20s",
		HeartbeatPeriod: "5s",
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
			SessionName:      "client",
		},
	})
	if err != nil {
		log.Error("hessian.NewClient(conf) = error:%s", jerrors.ErrorStack(err))
		return
	}
	clientInvoker, err = invoker.NewInvoker(clientRegistry,
		invoker.WithDubboClient(cltD),
		invoker.WithLBSelector(configClientLB))
}

func uninitClient() {
	log.Close()
}

func initProfiling(clientConfig *examples.ClientConfig) {
	if !clientConfig.Pprof_Enabled {
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
	addr = ip + ":" + strconv.Itoa(clientConfig.Pprof_Port)
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
			uninitClient()
			fmt.Println("app exit now...")
			return
		}
	}
}
