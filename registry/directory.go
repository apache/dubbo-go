package registry

import (
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/cluster/directory"
	"github.com/dubbo/dubbo-go/config"
)

type RegistryDirectory struct {
	cacheService *directory.ServiceArray
	listenerLock sync.Mutex
	serviceType  string
	registry     Registry
}

func NewRegistryDirectory(url config.RegistryURL, registry Registry) *RegistryDirectory {
	return &RegistryDirectory{
		cacheService: directory.NewServiceArray([]config.URL{}),
		serviceType:  url.URL.Service,
		registry:     registry,
	}
}

func (dir *RegistryDirectory) subscribe(url config.URL) {
	for {
		if dir.registry.IsClosed() {
			log.Warn("event listener game over.")
			return
		}

		listener, err := dir.registry.Subscribe(url)
		if err != nil {
			if dir.registry.IsClosed() {
				log.Warn("event listener game over.")
				return
			}
			log.Warn("getListener() = err:%s", jerrors.ErrorStack(err))
			time.Sleep(time.Duration(RegistryConnDelay) * time.Second)
			continue
		}

		for {
			if serviceEvent, err := listener.Next(); err != nil {
				log.Warn("Selector.watch() = error{%v}", jerrors.ErrorStack(err))
				listener.Close()
				time.Sleep(time.Duration(RegistryConnDelay) * time.Second)
				return
			} else {
				dir.update(serviceEvent)
			}

		}

	}
}

func (dir *RegistryDirectory) update(res *ServiceEvent) {
	if res == nil {
		return
	}

	log.Debug("registry update, result{%s}", res)
	registryKey := res.Service.Key()

	dir.listenerLock.Lock()
	defer dir.listenerLock.Unlock()

	svcArr, ok := dir.cacheService[registryKey]
	log.Debug("registry name:%s, its current member lists:%+v", registryKey, svcArr)

	switch res.Action {
	case ServiceAdd:
		if ok {
			svcArr.add(res.Service, ivk.ServiceTTL)
		} else {
			ivk.cacheServiceMap[registryKey] = newServiceArray([]registry.ServiceURL{res.Service})
		}
	case registry.ServiceDel:
		if ok {
			svcArr.del(res.Service, ivk.ServiceTTL)
			if len(svcArr.arr) == 0 {
				delete(ivk.cacheServiceMap, registryKey)
				log.Warn("delete registry %s from registry map", registryKey)
			}
		}
		log.Error("selector delete registryURL{%s}", res.Service)
	}
}
