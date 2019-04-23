package registry

import (
	"context"
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

type Options struct {
	serviceTTL time.Duration
}
type Option func(*Options)

func WithServiceTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.serviceTTL = ttl
	}
}

type RegistryDirectory struct {
	directory.BaseDirectory
	cacheService *directory.ServiceArray
	listenerLock sync.Mutex
	serviceType  string
	registry     Registry
	Options
}

func NewRegistryDirectory(ctx context.Context, url *config.RegistryURL, registry Registry, opts ...Option) *RegistryDirectory {
	options := Options{
		//default 300s
		serviceTTL: time.Duration(300e9),
	}
	for _, opt := range opts {
		opt(&options)
	}

	return &RegistryDirectory{
		BaseDirectory: directory.NewBaseDirectory(ctx, url),
		cacheService:  directory.NewServiceArray(ctx, []config.URL{}),
		serviceType:   url.URL.Service,
		registry:      registry,
		Options:       options,
	}
}

//subscibe from registry
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

//subscribe service from registry , and update the cacheServices
func (dir *RegistryDirectory) update(res *ServiceEvent) {
	if res == nil {
		return
	}

	log.Debug("registry update, result{%s}", res)

	dir.listenerLock.Lock()
	defer dir.listenerLock.Unlock()

	log.Debug("update service name: %s!", res.Service)

	switch res.Action {
	case ServiceAdd:
		dir.cacheService.Add(res.Service, dir.serviceTTL)

	case ServiceDel:
		dir.cacheService.Del(res.Service, dir.serviceTTL)

		log.Error("selector delete service url{%s}", res.Service)
	}
}

func (dir *RegistryDirectory) List() {

}
func (dir *RegistryDirectory) IsAvailable() bool {
	return true
}

func (dir *RegistryDirectory) Destroy() {
	dir.BaseDirectory.Destroy()
}
