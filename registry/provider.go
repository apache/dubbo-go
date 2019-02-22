package registry

import (
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/version"
)

//////////////////////////////////////////////
// ZkProviderRegistry
//////////////////////////////////////////////

const (
	ProviderRegistryZkClient = "provider zk registry"
)

type ProviderServiceConfig struct {
	ServiceConfig
	Path    string
	Methods string
}

func (c ProviderServiceConfig) String() string {
	return fmt.Sprintf(
		"%s@%s-%s-%s-%s/%s",
		c.ServiceConfig.Service,
		c.ServiceConfig.Protocol,
		c.ServiceConfig.Group,
		c.ServiceConfig.Version,
		c.Path,
		c.Methods,
	)
}

func (c ProviderServiceConfig) ServiceEqual(url *ServiceURL) bool {
	if c.ServiceConfig.Protocol != url.Protocol {
		return false
	}

	if c.ServiceConfig.Service != url.Query.Get("interface") {
		return false
	}

	if c.Group != "" && c.ServiceConfig.Group != url.Group {
		return false
	}

	if c.ServiceConfig.Version != "" && c.ServiceConfig.Version != url.Version {
		return false
	}

	if c.Path != url.Path {
		return false
	}

	if c.Methods != url.Query.Get("methods") {
		return false
	}

	return true
}

type ZkProviderRegistry struct {
	Options
	birth      int64          // time of file birth, seconds since Epoch; 0 if unknown
	wg         sync.WaitGroup // wg+done for zk restart
	done       chan struct{}
	sync.Mutex // lock for client + services
	client     *zookeeperClient
	services   map[string]ServiceConfigIf // service name + protocol -> service config
	zkPath     map[string]int             // key = protocol://ip:port/interface
}

func NewZkProviderRegistry(opts ...Option) (*ZkProviderRegistry, error) {
	r := &ZkProviderRegistry{
		birth:    time.Now().Unix(),
		done:     make(chan struct{}),
		services: make(map[string]ServiceConfigIf),
		zkPath:   make(map[string]int),
	}

	for _, o := range opts {
		o(&r.Options)
	}

	if r.Name == "" {
		r.Name = ProviderRegistryZkClient
	}
	if r.Version == "" {
		r.Version = version.Version
	}
	if r.RegistryConfig.Timeout == 0 {
		r.RegistryConfig.Timeout = DEFAULT_REGISTRY_TIMEOUT
	}
	err := r.validateZookeeperClient()
	if err != nil {
		return nil, jerrors.Trace(err)
	}
	r.wg.Add(1)
	go r.handleZkRestart()

	return r, nil
}

func (r *ZkProviderRegistry) validateZookeeperClient() error {
	var (
		err error
	)

	err = nil
	r.Lock()
	if r.client == nil {
		r.client, err = newZookeeperClient(ProviderRegistryZkClient, r.Address, r.RegistryConfig.Timeout)
		if err != nil {
			log.Warn("newZookeeperClient(name{%s}, zk addresss{%v}, timeout{%d}) = error{%#v}",
				ProviderRegistryZkClient, r.Address, r.Timeout, jerrors.ErrorStack(err))
		}
	}
	r.Unlock()

	return jerrors.Annotatef(err, "newZookeeperClient(ProviderRegistryZkClient, addr:%+v)", r.Address)
}

func (r *ZkProviderRegistry) Register(c interface{}) error {
	var (
		ok   bool
		err  error
		conf ProviderServiceConfig
	)

	if conf, ok = c.(ProviderServiceConfig); !ok {
		return jerrors.Errorf("@c{%v} type is not ServiceConfig", c)
	}

	// 检验服务是否已经注册过
	ok = false
	r.Lock()
	// 注意此处与consumerZookeeperRegistry的差异，consumer用的是conf.Service，
	// 因为consumer要提供watch功能给selector使用, provider允许注册同一个service的多个group or version
	_, ok = r.services[conf.String()]
	r.Unlock()
	if ok {
		return jerrors.Errorf("Service{%s} has been registered", conf.String())
	}

	err = r.register(&conf)
	if err != nil {
		return jerrors.Annotatef(err, "register(conf:%+v)", conf)
	}

	r.Lock()
	r.services[conf.String()] = &conf
	log.Debug("(ZkProviderRegistry)Register(conf{%#v})", conf)
	r.Unlock()

	return nil
}

func (r *ZkProviderRegistry) registerTempZookeeperNode(root string, node string) error {
	var (
		err    error
		zkPath string
	)

	r.Lock()
	defer r.Unlock()
	err = r.client.Create(root)
	if err != nil {
		log.Error("zk.Create(root{%s}) = err{%s}", root, jerrors.ErrorStack(err))
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

func (r *ZkProviderRegistry) register(conf *ProviderServiceConfig) error {
	var (
		err        error
		revision   string
		params     url.Values
		urlPath    string
		rawURL     string
		encodedURL string
		dubboPath  string
	)

	if conf.ServiceConfig.Service == "" || conf.Methods == "" {
		return jerrors.Errorf("conf{Service:%s, Methods:%s}", conf.ServiceConfig.Service, conf.Methods)
	}

	err = r.validateZookeeperClient()
	if err != nil {
		return jerrors.Trace(err)
	}
	// 先创建服务下面的provider node
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, DubboNodes[PROVIDER])
	r.Lock()
	err = r.client.Create(dubboPath)
	r.Unlock()
	if err != nil {
		log.Error("zkClient.create(path{%s}) = error{%#v}", dubboPath, jerrors.ErrorStack(err))
		return jerrors.Annotatef(err, "zkclient.Create(path:%s)", dubboPath)
	}

	params = url.Values{}
	params.Add("interface", conf.ServiceConfig.Service)
	params.Add("application", r.ApplicationConfig.Name)
	revision = r.ApplicationConfig.Version
	if revision == "" {
		revision = "0.1.0"
	}
	params.Add("revision", revision) // revision是pox.xml中application的version属性的值
	if conf.ServiceConfig.Group != "" {
		params.Add("group", conf.ServiceConfig.Group)
	}
	// dubbo java consumer来启动找provider url时，因为category不匹配，会找不到provider，导致consumer启动不了,所以使用consumers&providers
	// DubboRole               = [...]string{"consumer", "", "", "provider"}
	// params.Add("category", (DubboType(PROVIDER)).Role())
	params.Add("category", (DubboType(PROVIDER)).String())
	params.Add("dubbo", "dubbo-provider-golang-"+version.Version)
	params.Add("org", r.ApplicationConfig.Organization)
	params.Add("module", r.ApplicationConfig.Module)
	params.Add("owner", r.ApplicationConfig.Owner)
	params.Add("side", (DubboType(PROVIDER)).Role())
	params.Add("pid", processID)
	params.Add("ip", localIP)
	params.Add("timeout", fmt.Sprintf("%v", r.Timeout))
	// params.Add("timestamp", time.Now().Format("20060102150405"))
	params.Add("timestamp", fmt.Sprintf("%d", r.birth))
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
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, (DubboType(PROVIDER)).String())
	err = r.registerTempZookeeperNode(dubboPath, encodedURL)
	log.Debug("provider path:%s, url:%s", dubboPath, rawURL)
	if err != nil {
		return jerrors.Annotatef(err, "registerTempZookeeperNode(path:%s, url:%s)", dubboPath, rawURL)
	}

	return nil
}

func (r *ZkProviderRegistry) handleZkRestart() {
	var (
		err       error
		flag      bool
		failTimes int
		confIf    ServiceConfigIf
		services  []ServiceConfigIf
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
						err = r.register(confIf.(*ProviderServiceConfig))
						if err != nil {
							log.Error("(ZkProviderRegistry)register(conf{%#v}) = error{%#v}",
								confIf.(*ProviderServiceConfig), jerrors.ErrorStack(err))
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

func (r *ZkProviderRegistry) closeRegisters() {
	r.Lock()
	defer r.Unlock()
	log.Info("begin to close provider zk client")
	// 先关闭旧client，以关闭tmp node
	r.client.Close()
	r.client = nil
	r.services = nil
}

func (r *ZkProviderRegistry) Close() {
	close(r.done)
	r.wg.Wait()
	r.closeRegisters()
}
