package zookeeper

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
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
	"github.com/dubbo/dubbo-go/registry"
	"github.com/dubbo/dubbo-go/service"
	"github.com/dubbo/dubbo-go/version"
)

const (
	defaultTimeout           = int64(10e9)
	RegistryZkClient         = "zk registry"
	DEFAULT_REGISTRY_TIMEOUT = 1 * time.Second
	REGISTRY_CONN_DELAY      = 3
)

var (
	processID = ""
	localIP   = ""
)

type ZkRegistryConfig struct {
	Address    []string      `required:"true" yaml:"address"  json:"address,omitempty"`
	UserName   string        `yaml:"user_name" json:"user_name,omitempty"`
	Password   string        `yaml:"password" json:"password,omitempty"`
	TimeoutStr string        `yaml:"timeout" default:"5s" json:"timeout,omitempty"` // unit: second
	Timeout    time.Duration `yaml:"-"  json:"-"`
}

type Options struct {
	registry.Options
	ZkRegistryConfig
}

type Option func(*Options)

func (Option) OptionName() string {
	return "zk's option func"
}

type ZkRegistry struct {
	Options
	birth int64          // time of file birth, seconds since Epoch; 0 if unknown
	wg    sync.WaitGroup // wg+done for zk restart
	done  chan struct{}

	sync.Mutex
	client       *zookeeperClient
	services     map[string]service.ServiceConfigIf // service name + protocol -> service config
	listenerLock sync.Mutex
	listener     *zkEventListener
	//for provider
	zkPath       map[string]int // key = protocol://ip:port/interface
	outerEventCh chan *registry.ServiceURLEvent
}

func init() {
	processID = fmt.Sprintf("%d", os.Getpid())
	localIP, _ = gxnet.GetLocalIP()
}

func WithRegistryConf(conf ZkRegistryConfig) Option {
	return func(o *Options) {
		o.ZkRegistryConfig = conf
	}
}

func NewZkRegistry(opts ...registry.OptionInf) (registry.Registry, error) {
	var (
		err error
		r   *ZkRegistry
	)

	r = &ZkRegistry{
		birth:        time.Now().UnixNano(),
		done:         make(chan struct{}),
		services:     make(map[string]service.ServiceConfigIf),
		zkPath:       make(map[string]int),
		outerEventCh: make(chan *registry.ServiceURLEvent),
	}

	for _, opt := range opts {
		if o, ok := opt.(Option); ok {
			o(&r.Options)
		} else if o, ok := opt.(registry.Option); ok {
			o(&r.Options.Options)
		} else {
			return nil, jerrors.New("option is not available")
		}

	}
	//if r.DubboType == 0{
	//	return nil ,errors.New("Dubbo type should be specified.")
	//}
	if r.Name == "" {
		r.Name = RegistryZkClient
	}
	if r.Version == "" {
		r.Version = version.Version
	}

	if r.ZkRegistryConfig.Timeout == 0 {
		r.ZkRegistryConfig.Timeout = DEFAULT_REGISTRY_TIMEOUT
	}
	err = r.validateZookeeperClient()
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	r.wg.Add(1)
	go r.handleZkRestart()

	if r.DubboType == registry.CONSUMER {
		r.wg.Add(1)
		go r.listen()
	}

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
	r.Lock()
	defer r.Unlock()
	if r.client == nil {
		r.client, err = newZookeeperClient(RegistryZkClient, r.Address, r.ZkRegistryConfig.Timeout)
		if err != nil {
			log.Warn("newZookeeperClient(name{%s}, zk addresss{%v}, timeout{%d}) = error{%v}",
				RegistryZkClient, r.Address, r.Timeout.String(), err)
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

	return jerrors.Annotatef(err, "newZookeeperClient(address:%+v)", r.Address)
}

func (r *ZkRegistry) handleZkRestart() {
	var (
		err       error
		flag      bool
		failTimes int
		confIf    service.ServiceConfigIf
		services  []service.ServiceConfigIf
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
			r.Lock()
			r.client.Close()
			r.client = nil
			r.Unlock()

			// 接zk，直至成功
			failTimes = 0
			for {
				select {
				case <-r.done:
					log.Warn("(ZkProviderRegistry)reconnectZkRegistry goroutine exit now...")
					break LOOP
				case <-time.After(time.Duration(1e9 * failTimes * REGISTRY_CONN_DELAY)): // 防止疯狂重连zk
				}
				err = r.validateZookeeperClient()
				log.Info("ZkProviderRegistry.validateZookeeperClient(zkAddr{%s}) = error{%#v}",
					r.client.zkAddrs, jerrors.ErrorStack(err))
				if err == nil {
					// copy r.services
					r.Lock()
					for _, confIf = range r.services {
						services = append(services, confIf)
					}
					r.Unlock()

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
				if MAX_TIMES <= failTimes {
					failTimes = MAX_TIMES
				}
			}
		}
	}
}

func (r *ZkRegistry) register(c interface{}) error {
	var (
		err        error
		revision   string
		params     url.Values
		urlPath    string
		rawURL     string
		encodedURL string
		dubboPath  string
	)
	err = r.validateZookeeperClient()
	if err != nil {
		return jerrors.Trace(err)
	}
	params = url.Values{}

	params.Add("application", r.ApplicationConfig.Name)
	params.Add("default.timeout", fmt.Sprintf("%d", defaultTimeout/1e6))
	params.Add("environment", r.ApplicationConfig.Environment)
	params.Add("org", r.ApplicationConfig.Organization)
	params.Add("module", r.ApplicationConfig.Module)
	params.Add("owner", r.ApplicationConfig.Owner)
	params.Add("pid", processID)
	params.Add("ip", localIP)
	params.Add("timeout", fmt.Sprintf("%d", int64(r.Timeout)/1e6))
	params.Add("timestamp", fmt.Sprintf("%d", r.birth/1e6))

	revision = r.ApplicationConfig.Version
	if revision == "" {
		revision = "0.1.0"
	}
	params.Add("revision", revision) // revision是pox.xml中application的version属性的值

	if r.DubboType == registry.PROVIDER {

		conf := c.(ProviderServiceConfig)
		if conf.Service == "" || conf.Methods == "" {
			return jerrors.Errorf("conf{Service:%s, Methods:%s}", conf.Service, conf.Methods)
		}
		// 先创建服务下面的provider node
		dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, registry.DubboNodes[registry.PROVIDER])
		r.Lock()
		err = r.client.Create(dubboPath)
		r.Unlock()
		if err != nil {
			log.Error("zkClient.create(path{%s}) = error{%#v}", dubboPath, jerrors.ErrorStack(err))
			return jerrors.Annotatef(err, "zkclient.Create(path:%s)", dubboPath)
		}
		params.Add("anyhost", "true")
		params.Add("interface", conf.ServiceConfig.Service)

		if conf.ServiceConfig.Group != "" {
			params.Add("group", conf.ServiceConfig.Group)
		}
		// dubbo java consumer来启动找provider url时，因为category不匹配，会找不到provider，导致consumer启动不了,所以使用consumers&providers
		// DubboRole               = [...]string{"consumer", "", "", "provider"}
		// params.Add("category", (DubboType(PROVIDER)).Role())
		params.Add("category", (registry.DubboType(registry.PROVIDER)).String())
		params.Add("dubbo", "dubbo-provider-golang-"+version.Version)

		params.Add("side", (registry.DubboType(registry.PROVIDER)).Role())

		if conf.ServiceConfig.Version != "" {
			params.Add("version", conf.ServiceConfig.Version)
		}
		if conf.Methods != "" {
			params.Add("methods", conf.Methods)
		}
		log.Debug("provider zk url params:%#v", params)
		if conf.Path == "" {
			conf.Path = localIP
		}

		urlPath = conf.Service
		if r.zkPath[urlPath] != 0 {
			urlPath += strconv.Itoa(r.zkPath[urlPath])
		}
		r.zkPath[urlPath]++
		rawURL = fmt.Sprintf("%s://%s/%s?%s", conf.Protocol, conf.Path, urlPath, params.Encode())
		encodedURL = url.QueryEscape(rawURL)

		// 把自己注册service providers
		dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, (registry.DubboType(registry.PROVIDER)).String())
		log.Debug("provider path:%s, url:%s", dubboPath, rawURL)

	} else if r.DubboType == registry.CONSUMER {
		conf := c.(*service.ServiceConfig)
		dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, registry.DubboNodes[registry.CONSUMER])
		r.Lock()
		err = r.client.Create(dubboPath)
		r.Unlock()
		if err != nil {
			log.Error("zkClient.create(path{%s}) = error{%v}", dubboPath, jerrors.ErrorStack(err))
			return jerrors.Trace(err)
		}
		dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, registry.DubboNodes[registry.PROVIDER])
		r.Lock()
		err = r.client.Create(dubboPath)
		r.Unlock()
		if err != nil {
			log.Error("zkClient.create(path{%s}) = error{%v}", dubboPath, jerrors.ErrorStack(err))
			return jerrors.Trace(err)
		}

		params.Add("protocol", conf.Protocol)
		params.Add("interface", conf.Service)
		revision = r.ApplicationConfig.Version
		if revision == "" {
			revision = "0.1.0"
		}
		params.Add("revision", revision)
		if conf.Group != "" {
			params.Add("group", conf.Group)
		}
		params.Add("category", (registry.DubboType(registry.CONSUMER)).String())
		params.Add("dubbo", "dubbogo-consumer-"+version.Version)

		if conf.Version != "" {
			params.Add("version", conf.Version)
		}
		rawURL = fmt.Sprintf("consumer://%s/%s?%s", localIP, conf.Service+conf.Version, params.Encode())
		encodedURL = url.QueryEscape(rawURL)

		dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, (registry.DubboType(registry.CONSUMER)).String())
		log.Debug("consumer path:%s, url:%s", dubboPath, rawURL)
	} else {
		return jerrors.Errorf("@c{%v} type is not ServiceConfig or ProviderServiceConfig", c)
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

	r.Lock()
	defer r.Unlock()
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
	r.Lock()
	defer r.Unlock()
	log.Info("begin to close provider zk client")
	// 先关闭旧client，以关闭tmp node
	r.client.Close()
	r.client = nil
	r.services = nil
	//关闭outerListenerEvent
	close(r.outerEventCh)
}

func (r *ZkRegistry) isClosed() bool {
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}
