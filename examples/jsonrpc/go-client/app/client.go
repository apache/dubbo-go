package main

import (
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
	"github.com/AlexStocks/goext/net"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/client/invoker"
	"github.com/dubbo/dubbo-go/examples"
	"github.com/dubbo/dubbo-go/plugins"
	"github.com/dubbo/dubbo-go/public"
	"github.com/dubbo/dubbo-go/registry"
	"github.com/dubbo/dubbo-go/registry/zookeeper"
	"github.com/dubbo/dubbo-go/jsonrpc"
)

var (
	survivalTimeout int = 10e9
	clientInvokers  []*invoker.Invoker
)

func main() {
	clientConfig := examples.InitClientConfig()

	initProfiling(clientConfig)
	initClient(clientConfig)

	time.Sleep(10e9)

	log.Info("start to test jsonrpc")
	testJsonrpc("GetUser", "A003")
	log.Info("start to test jsonrpc illegal method")
	testJsonrpc("GetUser1", "A003")

	initSignal()
}

func initClient(clientConfig *examples.ClientConfig) {
	var (
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
		codecType = public.GetCodecType(clientConfig.Service_List[idx].Protocol())
		if codecType == public.CODECTYPE_UNKNOWN {
			panic(fmt.Sprintf("unknown protocol %s", clientConfig.Service_List[idx].Protocol()))
		}
	}

	for _, service := range clientConfig.Service_List {
		err = clientRegistry.Register(service)
		if err != nil {
			panic(fmt.Sprintf("registry.Register(service{%#v}) = error{%v}", service, jerrors.ErrorStack(err)))
			return
		}

		//read the client lb config in config.yml
		configClientLB := plugins.PluggableLoadbalance[clientConfig.ClientLoadBalance]()

		//init http client & init invoker
		clt := jsonrpc.NewHTTPClient(
			&jsonrpc.HTTPOptions{
				HandshakeTimeout: clientConfig.ConnectTimeout,
				HTTPTimeout:      clientConfig.RequestTimeout,
			},
		)

		clientInvoker, _ := invoker.NewInvoker(
			clientRegistry,
			service,
			invoker.WithHttpClient(clt),
			invoker.WithLBSelector(configClientLB),
		)

		clientInvokers = append(clientInvokers, clientInvoker)
	}
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

func testJsonrpc(method string, userKey string) {
	for _, clientInvoker := range clientInvokers {
		user := new(JsonRPCUser)
		err := clientInvoker.HttpCall(1, method, []string{userKey}, user)
		if err != nil {
			log.Error(err)
		} else {
			log.Info(user)
		}
	}
}
