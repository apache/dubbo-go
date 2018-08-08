package registry

import (
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/goext/net"
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/version"
)

const (
	REGISTRY_CONN_DELAY = 3
)

var (
	ErrorRegistryNotFound = jerrors.New("registry not found")
)

//////////////////////////////////////////////
// DubboType
//////////////////////////////////////////////

type DubboType int

const (
	CONSUMER = iota
	CONFIGURATOR
	ROUTER
	PROVIDER
)

var (
	DubboNodes       = [...]string{"consumers", "configurators", "routers", "providers"}
	DubboRole        = [...]string{"consumer", "", "", "provider"}
	RegistryZkClient = "zk registry"
	processID        = ""
	localIP          = ""
)

func init() {
	processID = fmt.Sprintf("%d", os.Getpid())
	localIP, _ = gxnet.GetLocalIP()
}

func (t DubboType) String() string {
	return DubboNodes[t]
}

func (t DubboType) Role() string {
	return DubboRole[t]
}

//////////////////////////////////////////////
// ZkConsumerRegistry
//////////////////////////////////////////////

const (
	DEFAULT_REGISTRY_TIMEOUT = 1
	ConsumerRegistryZkClient = "consumer zk registry"
)

type Options struct {
	ApplicationConfig
	RegistryConfig // ZooKeeperServers []string
	mode           Mode
	serviceTTL     time.Duration
}

type Option func(*Options)

func ApplicationConf(conf ApplicationConfig) Option {
	return func(o *Options) {
		o.ApplicationConfig = conf
	}
}

func RegistryConf(conf RegistryConfig) Option {
	return func(o *Options) {
		o.RegistryConfig = conf
	}
}

func BalanceMode(mode Mode) Option {
	return func(o *Options) {
		o.mode = mode
	}
}

func ServiceTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.serviceTTL = ttl
	}
}

type ZkConsumerRegistry struct {
	Options
	birth int64          // time of file birth, seconds since Epoch; 0 if unknown
	wg    sync.WaitGroup // wg+done for zk restart
	done  chan struct{}

	sync.Mutex
	client             *zookeeperClient
	services           map[string]ServiceConfigIf // service name + protocol -> service config
	listenerLock       sync.Mutex
	listener           *zkEventListener
	listenerServiceMap map[string]*serviceArray
}

func NewZkConsumerRegistry(opts ...Option) (*ZkConsumerRegistry, error) {
	var (
		err error
		r   *ZkConsumerRegistry
	)

	r = &ZkConsumerRegistry{
		birth:              time.Now().Unix(),
		done:               make(chan struct{}),
		services:           make(map[string]ServiceConfigIf),
		listenerServiceMap: make(map[string]*serviceArray),
	}

	for _, opt := range opts {
		opt(&r.Options)
	}

	if len(r.Name) == 0 {
		r.Name = ConsumerRegistryZkClient
	}
	if len(r.Version) == 0 {
		r.Version = version.Version
	}
	if r.RegistryConfig.Timeout == 0 {
		r.RegistryConfig.Timeout = DEFAULT_REGISTRY_TIMEOUT
	}
	err = r.validateZookeeperClient()
	if err != nil {
		return nil, jerrors.Trace(err)
	}

	r.wg.Add(1)
	go r.handleZkRestart()
	r.wg.Add(1)
	go r.listen()

	return r, nil
}

func (r *ZkConsumerRegistry) isClosed() bool {
	select {
	case <-r.done:
		return true
	default:
		return false
	}
}

func (r *ZkConsumerRegistry) handleZkRestart() {
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
			log.Warn("(consumerZkConsumerRegistry)reconnectZkRegistry goroutine exit now...")
			break LOOP
			// re-register all services
		case <-r.client.done():
			r.Lock()
			r.client.Close()
			r.client = nil
			r.Unlock()

			failTimes = 0
			for {
				select {
				case <-r.done:
					log.Warn("(consumerZkConsumerRegistry)reconnectZkRegistry goroutine exit now...")
					break LOOP
				case <-time.After(timeSecondDuration(failTimes * REGISTRY_CONN_DELAY)):
				}
				err = r.validateZookeeperClient()
				if err == nil {
					// copy r.services
					r.Lock()
					for _, confIf = range r.services {
						services = append(services, confIf)
					}
					r.Unlock()

					flag = true
					for _, confIf = range services {
						err = r.register(confIf.(*ServiceConfig))
						if err != nil {
							log.Error("in (consumerZkConsumerRegistry)reRegister, (consumerZkConsumerRegistry)register(conf{%#v}) = error{%#v}",
								confIf.(*ServiceConfig), jerrors.ErrorStack(err))
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

func (r *ZkConsumerRegistry) validateZookeeperClient() error {
	var (
		err error
	)

	err = nil
	r.Lock()
	defer r.Unlock()
	if r.client == nil {
		r.client, err = newZookeeperClient(ConsumerRegistryZkClient, r.Address, r.RegistryConfig.Timeout)
		if err != nil {
			log.Warn("newZookeeperClient(name{%s}, zk addresss{%v}, timeout{%d}) = error{%v}",
				ConsumerRegistryZkClient, r.Address, r.Timeout, err)
		}
	}

	return jerrors.Annotatef(err, "newZookeeperClient(address:%+v)", r.Address)
}

func (r *ZkConsumerRegistry) registerZookeeperNode(root string, data []byte) error {
	var (
		err    error
		zkPath string
	)

	r.Lock()
	defer r.Unlock()
	err = r.client.Create(root)
	if err != nil {
		log.Error("zk.Create(root{%s}) = err{%v}", root, jerrors.ErrorStack(err))
		return jerrors.Annotatef(err, "zkclient.Create(root:%s)", root)
	}
	zkPath, err = r.client.RegisterTempSeq(root, data)
	if err != nil {
		log.Error("createTempSeqNode(root{%s}) = error{%v}", root, jerrors.ErrorStack(err))
		return jerrors.Annotatef(err, "createTempSeqNode(root{%s})", root)
	}
	log.Debug("create a zookeeper node:%s", zkPath)

	return nil
}

func (r *ZkConsumerRegistry) registerTempZookeeperNode(root string, node string) error {
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

func (r *ZkConsumerRegistry) register(conf *ServiceConfig) error {
	var (
		err        error
		params     url.Values
		revision   string
		rawURL     string
		encodedURL string
		dubboPath  string
	)

	err = r.validateZookeeperClient()
	if err != nil {
		log.Error("client.validateZookeeperClient() = err:%#v", err)
		return jerrors.Trace(err)
	}
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, DubboNodes[CONSUMER])
	r.Lock()
	err = r.client.Create(dubboPath)
	r.Unlock()
	if err != nil {
		log.Error("zkClient.create(path{%s}) = error{%v}", dubboPath, jerrors.ErrorStack(err))
		return jerrors.Trace(err)
	}
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, DubboNodes[PROVIDER])
	r.Lock()
	err = r.client.Create(dubboPath)
	r.Unlock()
	if err != nil {
		log.Error("zkClient.create(path{%s}) = error{%v}", dubboPath, jerrors.ErrorStack(err))
		return jerrors.Trace(err)
	}

	params = url.Values{}
	params.Add("interface", conf.Service)
	params.Add("application", r.ApplicationConfig.Name)
	revision = r.ApplicationConfig.Version
	if revision == "" {
		revision = "0.1.0"
	}
	params.Add("revision", revision)
	if conf.Group != "" {
		params.Add("group", conf.Group)
	}
	params.Add("category", (DubboType(CONSUMER)).String())
	params.Add("dubbo", "dubbogo-consumer-"+version.Version)
	params.Add("org", r.Organization)
	params.Add("module", r.Module)
	params.Add("owner", r.Owner)
	params.Add("side", (DubboType(CONSUMER)).Role())
	params.Add("pid", processID)
	params.Add("ip", localIP)
	params.Add("timeout", fmt.Sprintf("%v", r.Timeout))
	params.Add("timestamp", fmt.Sprintf("%d", r.birth))
	if conf.Version != "" {
		params.Add("version", conf.Version)
	}
	rawURL = fmt.Sprintf("%s://%s/%s?%s", conf.Protocol, localIP, conf.Service+conf.Version, params.Encode())
	encodedURL = url.QueryEscape(rawURL)

	dubboPath = fmt.Sprintf("/dubbo/%s/%s", conf.Service, (DubboType(CONSUMER)).String())
	log.Debug("consumer path:%s, url:%s", dubboPath, rawURL)
	err = r.registerTempZookeeperNode(dubboPath, encodedURL)
	if err != nil {
		return jerrors.Trace(err)
	}

	return nil
}

func (r *ZkConsumerRegistry) Register(conf ServiceConfig) error {
	var (
		ok       bool
		err      error
		listener *zkEventListener
	)

	ok = false
	r.Lock()
	_, ok = r.services[conf.Key()]
	r.Unlock()
	if ok {
		return jerrors.Errorf("Service{%s} has been registered", conf.Service)
	}

	err = r.register(&conf)
	if err != nil {
		return jerrors.Trace(err)
	}

	r.Lock()
	r.services[conf.Key()] = &conf
	r.Unlock()
	log.Debug("(consumerZkConsumerRegistry)Register(conf{%#v})", conf)

	r.listenerLock.Lock()
	listener = r.listener
	r.listenerLock.Unlock()
	if listener != nil {
		go listener.listenServiceEvent(conf)
	}

	return nil
}

// name: service@protocol
func (r *ZkConsumerRegistry) get(sc ServiceConfig) ([]*ServiceURL, error) {
	var (
		ok            bool
		err           error
		dubboPath     string
		nodes         []string
		listener      *zkEventListener
		serviceURL    *ServiceURL
		serviceConfIf ServiceConfigIf
		serviceConf   *ServiceConfig
	)

	r.listenerLock.Lock()
	listener = r.listener
	r.listenerLock.Unlock()

	if listener != nil {
		listener.listenServiceEvent(sc)
	}

	r.Lock()
	serviceConfIf, ok = r.services[sc.Key()]
	r.Unlock()
	if !ok {
		return nil, jerrors.Errorf("Service{%s} has not been registered", sc.Key())
	}
	serviceConf, ok = serviceConfIf.(*ServiceConfig)
	if !ok {
		return nil, jerrors.Errorf("Service{%s}: failed to get serviceConfigIf type", sc.Key())
	}

	dubboPath = fmt.Sprintf("/dubbo/%s/providers", sc.Service)
	err = r.validateZookeeperClient()
	if err != nil {
		return nil, jerrors.Trace(err)
	}
	r.Lock()
	nodes, err = r.client.getChildren(dubboPath)
	r.Unlock()
	if err != nil {
		log.Warn("getChildren(dubboPath{%s}) = error{%v}", dubboPath, err)
		return nil, jerrors.Trace(err)
	}

	var listenerServiceMap = make(map[string]*ServiceURL)
	for _, n := range nodes {
		serviceURL, err = NewServiceURL(n)
		if err != nil {
			log.Error("NewServiceURL({%s}) = error{%v}", n, err)
			continue
		}
		if !serviceConf.ServiceEqual(serviceURL) {
			log.Warn("serviceURL{%s} is not compatible with ServiceConfig{%#v}", serviceURL, serviceConf)
			continue
		}

		_, ok := listenerServiceMap[serviceURL.Query.Get(serviceURL.Location)]
		if !ok {
			listenerServiceMap[serviceURL.Location] = serviceURL
			continue
		}
	}

	var services []*ServiceURL
	for _, service := range listenerServiceMap {
		services = append(services, service)
	}

	return services, nil
}

func (r *ZkConsumerRegistry) Filter(s ServiceConfigIf, reqID int64) (*ServiceURL, error) {
	var serviceConf ServiceConfig
	if scp, ok := s.(*ServiceConfig); ok {
		serviceConf = *scp
	} else if sc, ok := s.(ServiceConfig); ok {
		serviceConf = sc
	} else {
		return nil, jerrors.Errorf("illegal @s:%#v", s)
	}

	serviceKey := serviceConf.Key()

	r.listenerLock.Lock()
	svcArr, sok := r.listenerServiceMap[serviceKey]
	log.Debug("r.svcArr[serviceString{%v}] = svcArr{%s}", serviceKey, svcArr)
	if sok {
		if serviceURL, err := svcArr.Select(reqID, r.Options.mode, r.Options.serviceTTL); err == nil {
			r.listenerLock.Unlock()
			return serviceURL, nil
		}
	}
	r.listenerLock.Unlock()

	svcs, err := r.get(serviceConf)
	r.listenerLock.Lock()
	defer r.listenerLock.Unlock()
	if err != nil {
		log.Error("Registry.get(conf:%+v) = {err:%r, svcs:%+v}",
			serviceConf, jerrors.ErrorStack(err), svcs)
		if sok && len(svcArr.arr) > 0 {
			log.Error("serviceArray{%v} timeout, can not get new, use old instead", svcArr)
			service, err := svcArr.Select(reqID, r.Options.mode, 0)
			return service, jerrors.Trace(err)
		}

		return nil, jerrors.Trace(err)
	}

	newSvcArr := newServiceArray(svcs)
	service, err := newSvcArr.Select(reqID, r.Options.mode, 0)
	r.listenerServiceMap[serviceKey] = newSvcArr
	return service, jerrors.Trace(err)
}

func (r *ZkConsumerRegistry) getListener() (*zkEventListener, error) {
	var (
		ok          bool
		zkListener  *zkEventListener
		serviceConf *ServiceConfig
	)

	r.listenerLock.Lock()
	zkListener = r.listener
	r.listenerLock.Unlock()
	if zkListener != nil {
		return zkListener, nil
	}

	r.Lock()
	client := r.client
	r.Unlock()
	if client == nil {
		return nil, jerrors.New("zk connection broken")
	}

	// new client & listener
	zkListener = newZkEventListener(client)

	r.listenerLock.Lock()
	r.listener = zkListener
	r.listenerLock.Unlock()

	// listen
	r.Lock()
	for _, service := range r.services {
		if serviceConf, ok = service.(*ServiceConfig); ok {
			go zkListener.listenServiceEvent(*serviceConf)
		}
	}
	r.Unlock()

	return zkListener, nil
}

func (r *ZkConsumerRegistry) update(res *ServiceURLEvent) {
	if res == nil || res.Service == nil {
		return
	}

	log.Debug("registry update, result{%s}", res)
	serviceKey := res.Service.ServiceConfig().Key()

	r.listenerLock.Lock()
	defer r.listenerLock.Unlock()

	svcArr, ok := r.listenerServiceMap[serviceKey]
	log.Debug("service name:%s, its current member lists:%+v", serviceKey, svcArr)

	switch res.Action {
	case ServiceURLAdd:
		if ok {
			svcArr.Add(res.Service, r.Options.serviceTTL)
		} else {
			r.listenerServiceMap[serviceKey] = newServiceArray([]*ServiceURL{res.Service})
		}
	case ServiceURLDel:
		if ok {
			svcArr.Del(res.Service, r.Options.serviceTTL)
			if len(svcArr.arr) == 0 {
				delete(r.listenerServiceMap, serviceKey)
				log.Warn("delete service %s from service map", serviceKey)
			}
		}
		log.Error("selector delete serviceURL{%s}", *res.Service)
	}
}

func (r *ZkConsumerRegistry) listen() {
	defer r.wg.Done()

	for {
		if r.isClosed() {
			log.Warn("event listener game over.")
			return
		}

		listener, err := r.getListener()
		if err != nil {
			if r.isClosed() {
				log.Warn("event listener game over.")
				return
			}
			log.Warn("getListener() = err:%s", jerrors.ErrorStack(err))
			time.Sleep(timeSecondDuration(REGISTRY_CONN_DELAY))
			continue
		}

		if err = listener.listenEvent(r); err != nil {
			log.Warn("Selector.watch() = error{%v}", jerrors.ErrorStack(err))

			r.listenerLock.Lock()
			r.listener = nil
			r.listenerLock.Unlock()

			listener.close()

			time.Sleep(timeSecondDuration(REGISTRY_CONN_DELAY))
			continue
		}
	}
}

func (r *ZkConsumerRegistry) closeRegisters() {
	r.Lock()
	log.Info("begin to close zk client")
	r.client.Close()
	r.client = nil
	r.services = nil
	r.Unlock()
}

func (r *ZkConsumerRegistry) Close() {
	close(r.done)
	r.wg.Wait()
	r.closeRegisters()
}
