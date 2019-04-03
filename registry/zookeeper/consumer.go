package zookeeper

import (
	"fmt"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/registry"
)

// name: service@protocol
func (r *ZkRegistry) GetService(conf registry.ServiceConfig) ([]*registry.ServiceURL, error) {
	var (
		ok            bool
		err           error
		dubboPath     string
		nodes         []string
		listener      *zkEventListener
		serviceURL    *registry.ServiceURL
		serviceConfIf registry.ServiceConfigIf
		serviceConf   registry.ServiceConfig
	)
	r.listenerLock.Lock()
	listener = r.listener
	r.listenerLock.Unlock()

	if listener != nil {
		listener.listenServiceEvent(conf)
	}

	r.cltLock.Lock()
	serviceConfIf, ok = r.services[conf.Key()]
	r.cltLock.Unlock()
	if !ok {
		return nil, jerrors.Errorf("Service{%s} has not been registered", conf.Key())
	}
	serviceConf, ok = serviceConfIf.(registry.ServiceConfig)
	if !ok {
		return nil, jerrors.Errorf("Service{%s}: failed to get serviceConfigIf type", conf.Key())
	}

	dubboPath = fmt.Sprintf("/dubbo/%s/providers", conf.Service)
	err = r.validateZookeeperClient()
	if err != nil {
		return nil, jerrors.Trace(err)
	}
	r.cltLock.Lock()
	nodes, err = r.client.getChildren(dubboPath)
	r.cltLock.Unlock()
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

func (r *ZkRegistry) Subscribe() (registry.Listener, error) {
	r.wg.Add(1)
	return r.getListener()
}

func (r *ZkRegistry) getListener() (*zkEventListener, error) {
	var (
		ok          bool
		zkListener  *zkEventListener
		serviceConf registry.ServiceConfig
	)

	r.listenerLock.Lock()
	zkListener = r.listener
	r.listenerLock.Unlock()
	if zkListener != nil {
		return zkListener, nil
	}

	r.cltLock.Lock()
	client := r.client
	r.cltLock.Unlock()
	if client == nil {
		return nil, jerrors.New("zk connection broken")
	}

	// new client & listener
	zkListener = newZkEventListener(r, client)

	r.listenerLock.Lock()
	r.listener = zkListener
	r.listenerLock.Unlock()

	// listen
	r.cltLock.Lock()
	for _, svs := range r.services {
		if serviceConf, ok = svs.(registry.ServiceConfig); ok {
			go zkListener.listenServiceEvent(serviceConf)
		}
	}
	r.cltLock.Unlock()

	return zkListener, nil
}
