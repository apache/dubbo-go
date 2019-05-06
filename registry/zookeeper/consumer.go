package zookeeper

import (
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/go-for-apache-dubbo/config"
	"github.com/dubbo/go-for-apache-dubbo/registry"
)

// name: service@protocol
//func (r *ZkRegistry) GetService(conf registry.ReferenceConfig) ([]config.SubURL, error) {
//
//	var (
//		err         error
//		dubboPath   string
//		nodes       []string
//		listener    *zkEventListener
//		serviceURL  config.SubURL
//		serviceConf registry.ReferenceConfig
//		ok          bool
//	)
//	r.listenerLock.Lock()
//	listener = r.listener
//	r.listenerLock.Unlock()
//
//	if listener != nil {
//		listener.listenServiceEvent(conf)
//	}
//
//	r.cltLock.Lock()
//	serviceConf, ok = r.services[conf.Key()]
//	r.cltLock.Unlock()
//	if !ok {
//		return nil, jerrors.Errorf("Path{%s} has not been registered", conf.Key())
//	}
//	if !ok {
//		return nil, jerrors.Errorf("Path{%s}: failed to get serviceConfigIf type", conf.Key())
//	}
//
//	dubboPath = fmt.Sprintf("/dubbo/%s/providers", conf.Path())
//	err = r.validateZookeeperClient()
//	if err != nil {
//		return nil, jerrors.Trace(err)
//	}
//	r.cltLock.Lock()
//	nodes, err = r.client.getChildren(dubboPath)
//	r.cltLock.Unlock()
//	if err != nil {
//		log.Warn("getChildren(dubboPath{%s}) = error{%v}", dubboPath, err)
//		return nil, jerrors.Trace(err)
//	}
//
//	var listenerServiceMap = make(map[string]config.SubURL)
//	for _, n := range nodes {
//
//		serviceURL, err = plugins.DefaultServiceURL()(n)
//		if err != nil {
//			log.Error("NewURL({%s}) = error{%v}", n, err)
//			continue
//		}
//		if !serviceConf.ServiceEqual(serviceURL) {
//			log.Warn("serviceURL{%s} is not compatible with ReferenceConfig{%#v}", serviceURL, serviceConf)
//			continue
//		}
//
//		_, ok := listenerServiceMap[serviceURL.Params().Get(serviceURL.Location())]
//		if !ok {
//			listenerServiceMap[serviceURL.Location()] = serviceURL
//			continue
//		}
//	}
//
//	var services []config.SubURL
//	for _, service := range listenerServiceMap {
//		services = append(services, service)
//	}
//
//	return services, nil
//}

func (r *ZkRegistry) Subscribe(conf config.URL) (registry.Listener, error) {
	r.wg.Add(1)
	return r.getListener(conf)
}

func (r *ZkRegistry) getListener(conf config.URL) (*zkEventListener, error) {
	var (
		zkListener *zkEventListener
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
		if svs.URLEqual(conf) {
			go zkListener.listenServiceEvent(svs)
		}
	}
	r.cltLock.Unlock()

	return zkListener, nil
}
