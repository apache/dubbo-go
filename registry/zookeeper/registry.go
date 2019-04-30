package zookeeper

import (
	"context"
	"fmt"
	"github.com/dubbo/dubbo-go/common/constant"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/goext/net"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
	"github.com/samuel/go-zookeeper/zk"
)

import (
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/registry"
	"github.com/dubbo/dubbo-go/version"
)

const (
	defaultTimeout    = int64(10e9)
	RegistryZkClient  = "zk registry"
	RegistryConnDelay = 3
)

var (
	processID = ""
	localIP   = ""
)

func init() {
	processID = fmt.Sprintf("%d", os.Getpid())
	localIP, _ = gxnet.GetLocalIP()
	//plugins.PluggableRegistries["zookeeper"] = NewZkRegistry
	extension.SetRegistry("zookeeper", NewZkRegistry)
}

/////////////////////////////////////
// zookeeper registry
/////////////////////////////////////

type ZkRegistry struct {
	context context.Context
	*config.URL
	birth int64          // time of file birth, seconds since Epoch; 0 if unknown
	wg    sync.WaitGroup // wg+done for zk restart
	done  chan struct{}

	cltLock  sync.Mutex
	client   *zookeeperClient
	services map[string]config.URL // service name + protocol -> service config

	listenerLock sync.Mutex
	listener     *zkEventListener

	//for provider
	zkPath map[string]int // key = protocol://ip:port/interface
}

func NewZkRegistry(url *config.URL) (registry.Registry, error) {
	var (
		err error
		r   *ZkRegistry
	)

	r = &ZkRegistry{
		URL:      url,
		birth:    time.Now().UnixNano(),
		done:     make(chan struct{}),
		services: make(map[string]config.URL),
		zkPath:   make(map[string]int),
	}

	//if r.SubURL.Name == "" {
	//	r.SubURL.Name = RegistryZkClient
	//}
	//if r.Version == "" {
	//	r.Version = version.Version
	//}

	if r.Timeout == 0 {
		r.Timeout = 1e9
	}
	err = r.validateZookeeperClient()
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	r.wg.Add(1)
	go r.handleZkRestart()

	//if r.RoleType == registry.CONSUMER {
	//	r.wg.Add(1)
	//	go r.listen()
	//}

	return r, nil
}

func (r *ZkRegistry) Close() {
	close(r.done)
	r.wg.Wait()
	r.closeRegisters()
}

func (r *ZkRegistry) validateZookeeperClient() error {
	var (
		err error
	)

	err = nil
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	if r.client == nil {
		//in dubbp ,every registry only connect one node ,so this is []string{r.Address}
		r.client, err = newZookeeperClient(RegistryZkClient, []string{r.PrimitiveURL}, r.Timeout)
		if err != nil {
			log.Warn("newZookeeperClient(name{%s}, zk addresss{%v}, timeout{%d}) = error{%v}",
				RegistryZkClient, r.PrimitiveURL, r.Timeout.String(), err)
			return jerrors.Annotatef(err, "newZookeeperClient(address:%+v)", r.PrimitiveURL)
		}
	}
	if r.client.conn == nil {
		var event <-chan zk.Event
		r.client.conn, event, err = zk.Connect(r.client.zkAddrs, r.client.timeout)
		if err != nil {
			r.client.wait.Add(1)
			go r.client.handleZkEvent(event)
		}
	}

	return jerrors.Annotatef(err, "newZookeeperClient(address:%+v)", r.PrimitiveURL)
}

func (r *ZkRegistry) handleZkRestart() {
	var (
		err       error
		flag      bool
		failTimes int
		confIf    config.URL
		services  []config.URL
	)

	defer r.wg.Done()
LOOP:
	for {
		select {
		case <-r.done:
			log.Warn("(ZkProviderRegistry)reconnectZkRegistry goroutine exit now...")
			break LOOP
			// re-register all services
		case <-r.client.done():
			r.cltLock.Lock()
			r.client.Close()
			r.client = nil
			r.cltLock.Unlock()

			// 接zk，直至成功
			failTimes = 0
			for {
				select {
				case <-r.done:
					log.Warn("(ZkProviderRegistry)reconnectZkRegistry goroutine exit now...")
					break LOOP
				case <-time.After(time.Duration(1e9 * failTimes * RegistryConnDelay)): // 防止疯狂重连zk
				}
				err = r.validateZookeeperClient()
				log.Info("ZkProviderRegistry.validateZookeeperClient(zkAddr{%s}) = error{%#v}",
					r.client.zkAddrs, jerrors.ErrorStack(err))
				if err == nil {
					// copy r.services
					r.cltLock.Lock()
					for _, confIf = range r.services {
						services = append(services, confIf)
					}
					r.cltLock.Unlock()

					flag = true
					for _, confIf = range services {
						err = r.register(confIf)
						if err != nil {
							log.Error("(ZkProviderRegistry)register(conf{%#v}) = error{%#v}",
								confIf, jerrors.ErrorStack(err))
							flag = false
							break
						}
					}
					if flag {
						break
					}
				}
				failTimes++
				if MaxFailTimes <= failTimes {
					failTimes = MaxFailTimes
				}
			}
		}
	}
}

func (r *ZkRegistry) Register(conf config.URL) error {
	var (
		ok       bool
		err      error
		listener *zkEventListener
	)
	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	switch role {
	case config.CONSUMER:
		ok = false
		r.cltLock.Lock()
		_, ok = r.services[conf.Key()]
		r.cltLock.Unlock()
		if ok {
			return jerrors.Errorf("Service{%s} has been registered", conf.Service)
		}

		err = r.register(conf)
		if err != nil {
			return jerrors.Trace(err)
		}

		r.cltLock.Lock()
		r.services[conf.Key()] = conf
		r.cltLock.Unlock()
		log.Debug("(consumerZkConsumerRegistry)Register(conf{%#v})", conf)

		r.listenerLock.Lock()
		listener = r.listener
		r.listenerLock.Unlock()
		if listener != nil {
			go listener.listenServiceEvent(conf)
		}
	case config.PROVIDER:

		// 检验服务是否已经注册过
		ok = false
		r.cltLock.Lock()
		// 注意此处与consumerZookeeperRegistry的差异，consumer用的是conf.Service，
		// 因为consumer要提供watch功能给selector使用, provider允许注册同一个service的多个group or version
		_, ok = r.services[conf.String()]
		r.cltLock.Unlock()
		if ok {
			return jerrors.Errorf("Service{%s} has been registered", conf.String())
		}

		err = r.register(conf)
		if err != nil {
			return jerrors.Annotatef(err, "register(conf:%+v)", conf)
		}

		r.cltLock.Lock()
		r.services[conf.String()] = conf
		r.cltLock.Unlock()

		log.Debug("(ZkProviderRegistry)Register(conf{%#v})", conf)
	}

	return nil
}

func (r *ZkRegistry) register(c config.URL) error {
	var (
		err error
		//revision   string
		params     url.Values
		urlPath    string
		rawURL     string
		encodedURL string
		dubboPath  string
		conf       config.URL
	)

	err = r.validateZookeeperClient()
	if err != nil {
		return jerrors.Trace(err)
	}
	params = url.Values{}
	params = c.Params

	params.Add("pid", processID)
	params.Add("ip", localIP)
	params.Add("timeout", fmt.Sprintf("%d", int64(r.Timeout)/1e6))

	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	switch role {

	case config.PROVIDER:

		if conf.Service == "" || len(conf.Methods) == 0 {
			return jerrors.Errorf("conf{Service:%s, Methods:%s}", conf.Service, conf.Methods)
		}
		// 先创建服务下面的provider node
		dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, config.DubboNodes[config.PROVIDER])
		r.cltLock.Lock()
		err = r.client.Create(dubboPath)
		r.cltLock.Unlock()
		if err != nil {
			log.Error("zkClient.create(path{%s}) = error{%#v}", dubboPath, jerrors.ErrorStack(err))
			return jerrors.Annotatef(err, "zkclient.Create(path:%s)", dubboPath)
		}
		params.Add("anyhost", "true")
		params.Add("interface", conf.Service)

		// dubbo java consumer来启动找provider url时，因为category不匹配，会找不到provider，导致consumer启动不了,所以使用consumers&providers
		// DubboRole               = [...]string{"consumer", "", "", "provider"}
		// params.Add("category", (RoleType(PROVIDER)).Role())
		params.Add("category", (config.RoleType(config.PROVIDER)).String())
		params.Add("dubbo", "dubbo-provider-golang-"+version.Version)

		params.Add("side", (config.RoleType(config.PROVIDER)).Role())

		if len(conf.Methods) == 0 {
			params.Add("methods", strings.Join(conf.Methods, ","))
		}
		log.Debug("provider zk url params:%#v", params)
		var path = conf.Path
		if path == "" {
			path = localIP
		}

		urlPath = conf.Service
		if r.zkPath[urlPath] != 0 {
			urlPath += strconv.Itoa(r.zkPath[urlPath])
		}
		r.zkPath[urlPath]++
		rawURL = fmt.Sprintf("%s://%s/%s?%s", conf.Protocol, path, urlPath, params.Encode())
		encodedURL = url.QueryEscape(rawURL)

		// 把自己注册service providers
		dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, (config.RoleType(config.PROVIDER)).String())
		log.Debug("provider path:%s, url:%s", dubboPath, rawURL)

	case config.CONSUMER:
		dubboPath = fmt.Sprintf("/dubbo/%s/%s", c.Service, config.DubboNodes[config.CONSUMER])
		r.cltLock.Lock()
		err = r.client.Create(dubboPath)
		r.cltLock.Unlock()
		if err != nil {
			log.Error("zkClient.create(path{%s}) = error{%v}", dubboPath, jerrors.ErrorStack(err))
			return jerrors.Trace(err)
		}
		dubboPath = fmt.Sprintf("/dubbo/%s/%s", c.Service, config.DubboNodes[config.PROVIDER])
		r.cltLock.Lock()
		err = r.client.Create(dubboPath)
		r.cltLock.Unlock()
		if err != nil {
			log.Error("zkClient.create(path{%s}) = error{%v}", dubboPath, jerrors.ErrorStack(err))
			return jerrors.Trace(err)
		}

		params.Add("protocol", c.Protocol)
		params.Add("interface", c.Service)

		params.Add("category", (config.RoleType(config.CONSUMER)).String())
		params.Add("dubbo", "dubbogo-consumer-"+version.Version)

		rawURL = fmt.Sprintf("consumer://%s/%s?%s", localIP, c.Service, params.Encode())
		encodedURL = url.QueryEscape(rawURL)

		dubboPath = fmt.Sprintf("/dubbo/%s/%s", c.Service, (config.RoleType(config.CONSUMER)).String())
		log.Debug("consumer path:%s, url:%s", dubboPath, rawURL)
	default:
		return jerrors.Errorf("@c{%v} type is not referencer or provider", c)
	}

	err = r.registerTempZookeeperNode(dubboPath, encodedURL)

	if err != nil {
		return jerrors.Annotatef(err, "registerTempZookeeperNode(path:%s, url:%s)", dubboPath, rawURL)
	}
	return nil
}

func (r *ZkRegistry) registerTempZookeeperNode(root string, node string) error {
	var (
		err    error
		zkPath string
	)

	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	err = r.client.Create(root)
	if err != nil {
		log.Error("zk.Create(root{%s}) = err{%v}", root, jerrors.ErrorStack(err))
		return jerrors.Trace(err)
	}
	zkPath, err = r.client.RegisterTemp(root, node)
	if err != nil {
		log.Error("RegisterTempNode(root{%s}, node{%s}) = error{%v}", root, node, jerrors.ErrorStack(err))
		return jerrors.Annotatef(err, "RegisterTempNode(root{%s}, node{%s})", root, node)
	}
	log.Debug("create a zookeeper node:%s", zkPath)

	return nil
}

func (r *ZkRegistry) closeRegisters() {
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	log.Info("begin to close provider zk client")
	// 先关闭旧client，以关闭tmp node
	r.client.Close()
	r.client = nil
	r.services = nil
}

func (r *ZkRegistry) IsClosed() bool {
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}
