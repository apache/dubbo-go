package zookeeper

import (
	"fmt"
	"github.com/dubbo/dubbo-go/registry"
	"github.com/dubbo/dubbo-go/service"
	jerrors "github.com/juju/errors"
	log "github.com/AlexStocks/log4go"
	"time"
)

func (r *ZkRegistry) ConsumerRegister(conf *service.ServiceConfig) error {
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

	err = r.register(conf)
	if err != nil {
		return jerrors.Trace(err)
	}

	r.Lock()
	r.services[conf.Key()] = conf
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


func (r *ZkRegistry) Listen()chan *registry.ServiceURLEvent {
	eventCh := make(chan *registry.ServiceURLEvent,1000)
	go r.listen(eventCh)
	return eventCh
}


// name: service@protocol
func (r *ZkRegistry) GetService(conf *service.ServiceConfig) ([]*service.ServiceURL, error) {
	var (
		ok            bool
		err           error
		dubboPath     string
		nodes         []string
		listener      *zkEventListener
		serviceURL    *service.ServiceURL
		serviceConfIf service.ServiceConfigIf
		serviceConf   *service.ServiceConfig
	)
	r.listenerLock.Lock()
	listener = r.listener
	r.listenerLock.Unlock()

	if listener != nil {
		listener.listenServiceEvent(conf)
	}

	r.Lock()
	serviceConfIf, ok = r.services[conf.Key()]
	r.Unlock()
	if !ok {
		return nil, jerrors.Errorf("Service{%s} has not been registered", conf.Key())
	}
	serviceConf, ok = serviceConfIf.(*service.ServiceConfig)
	if !ok {
		return nil, jerrors.Errorf("Service{%s}: failed to get serviceConfigIf type", conf.Key())
	}

	dubboPath = fmt.Sprintf("/dubbo/%s/providers", conf.Service)
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

	var listenerServiceMap = make(map[string]*service.ServiceURL)
	for _, n := range nodes {
		serviceURL, err = service.NewServiceURL(n)
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

	var services []*service.ServiceURL
	for _, service := range listenerServiceMap {
		services = append(services, service)
	}

	return services, nil
}

func (r *ZkRegistry) listen(ch chan *registry.ServiceURLEvent) {
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

		if err = listener.listenEvent(r,ch); err != nil {
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

func (r *ZkRegistry) getListener() (*zkEventListener, error) {
	var (
		ok          bool
		zkListener  *zkEventListener
		serviceConf *service.ServiceConfig
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
	for _, svs := range r.services {
		if serviceConf, ok = svs.(*service.ServiceConfig); ok {
			go zkListener.listenServiceEvent(serviceConf)
		}
	}
	r.Unlock()

	return zkListener, nil
}

