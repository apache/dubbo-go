package zookeeper

import (
	"fmt"
	"github.com/dubbo/dubbo-go/registry"
	jerrors "github.com/juju/errors"
	log "github.com/AlexStocks/log4go"
	"time"
)

func (r *ZkRegistry) ConsumerRegister(conf registry.ServiceConfig) error {
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


func (r *ZkRegistry) Listen() {
	go r.listen()
}

func (r *ZkRegistry) Filter(s registry.ServiceConfigIf, reqID int64) (*registry.ServiceURL, error) {
	var serviceConf registry.ServiceConfig
	if scp, ok := s.(*registry.ServiceConfig); ok {
		serviceConf = *scp
	} else if sc, ok := s.(registry.ServiceConfig); ok {
		serviceConf = sc
	} else {
		return nil, jerrors.Errorf("illegal @s:%#v", s)
	}

	serviceKey := serviceConf.Key()

	r.listenerLock.Lock()
	svcArr, sok := r.listenerServiceMap[serviceKey]
	log.Debug("r.svcArr[serviceString{%v}] = svcArr{%s}", serviceKey, svcArr)
	if sok {
		if serviceURL, err := svcArr.Select(reqID, r.Mode, r.ServiceTTL); err == nil {
			r.listenerLock.Unlock()
			return serviceURL, nil
		}
	}
	r.listenerLock.Unlock()

	svcs, err := r.get(serviceConf)
	r.listenerLock.Lock()
	defer r.listenerLock.Unlock()
	if err != nil {
		log.Error("Registry.get(conf:%+v) = {err:%s, svcs:%+v}",
			serviceConf, jerrors.ErrorStack(err), svcs)
		if sok && len(svcArr.Arr) > 0 {
			log.Error("serviceArray{%v} timeout, can not get new, use old instead", svcArr)
			service, err := svcArr.Select(reqID, r.Mode, 0)
			return service, jerrors.Trace(err)
		}

		return nil, jerrors.Trace(err)
	}

	newSvcArr := registry.NewServiceArray(svcs)
	service, err := newSvcArr.Select(reqID, r.Mode, 0)
	r.listenerServiceMap[serviceKey] = newSvcArr
	return service, jerrors.Trace(err)
}


// name: service@protocol
func (r *ZkRegistry) get(sc registry.ServiceConfig) ([]*registry.ServiceURL, error) {
	var (
		ok            bool
		err           error
		dubboPath     string
		nodes         []string
		listener      *zkEventListener
		serviceURL    *registry.ServiceURL
		serviceConfIf registry.ServiceConfigIf
		serviceConf   *registry.ServiceConfig
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
	serviceConf, ok = serviceConfIf.(*registry.ServiceConfig)
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

	var listenerServiceMap = make(map[string]*registry.ServiceURL)
	for _, n := range nodes {
		serviceURL, err = registry.NewServiceURL(n)
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

	var services []*registry.ServiceURL
	for _, service := range listenerServiceMap {
		services = append(services, service)
	}

	return services, nil
}

func (r *ZkRegistry) listen() {
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

func (r *ZkRegistry) getListener() (*zkEventListener, error) {
	var (
		ok          bool
		zkListener  *zkEventListener
		serviceConf *registry.ServiceConfig
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
		if serviceConf, ok = service.(*registry.ServiceConfig); ok {
			go zkListener.listenServiceEvent(*serviceConf)
		}
	}
	r.Unlock()

	return zkListener, nil
}


func (r *ZkRegistry) update(res *registry.ServiceURLEvent) {
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
	case registry.ServiceURLAdd:
		if ok {
			svcArr.Add(res.Service, r.Options.ServiceTTL)
		} else {
			r.listenerServiceMap[serviceKey] = registry.NewServiceArray([]*registry.ServiceURL{res.Service})
		}
	case registry.ServiceURLDel:
		if ok {
			svcArr.Del(res.Service, r.Options.ServiceTTL)
			if len(svcArr.Arr) == 0 {
				delete(r.listenerServiceMap, serviceKey)
				log.Warn("delete service %s from service map", serviceKey)
			}
		}
		log.Error("selector delete serviceURL{%s}", *res.Service)
	}
}