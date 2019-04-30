package directory

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
	"github.com/dubbo/dubbo-go/common/constant"
	"github.com/dubbo/dubbo-go/common/extension"
	"github.com/dubbo/dubbo-go/config"
	"github.com/dubbo/dubbo-go/protocol"
	"github.com/dubbo/dubbo-go/protocol/protocolwrapper"
	"github.com/dubbo/dubbo-go/registry"
	protocol2 "github.com/dubbo/dubbo-go/registry/protocol"
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
	cacheInvokers    []protocol.Invoker
	listenerLock     sync.Mutex
	serviceType      string
	registry         registry.Registry
	cacheInvokersMap *sync.Map //use sync.map
	//cacheInvokersMap map[string]protocol.Invoker
	Options
}

func NewRegistryDirectory(url *config.URL, registry registry.Registry, opts ...Option) *RegistryDirectory {
	options := Options{
		//default 300s
		serviceTTL: time.Duration(300e9),
	}
	for _, opt := range opts {
		opt(&options)
	}

	return &RegistryDirectory{
		BaseDirectory:    directory.NewBaseDirectory(url),
		cacheInvokers:    []protocol.Invoker{},
		cacheInvokersMap: &sync.Map{},
		serviceType:      url.SubURL.Service,
		registry:         registry,
		Options:          options,
	}
}

//subscibe from registry
func (dir *RegistryDirectory) Subscribe(url config.URL) {
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
			time.Sleep(time.Duration(protocol2.RegistryConnDelay) * time.Second)
			continue
		}

		for {
			if serviceEvent, err := listener.Next(); err != nil {
				log.Warn("Selector.watch() = error{%v}", jerrors.ErrorStack(err))
				listener.Close()
				time.Sleep(time.Duration(protocol2.RegistryConnDelay) * time.Second)
				return
			} else {
				go dir.update(serviceEvent)
			}

		}

	}
}

//subscribe service from registry , and update the cacheServices
func (dir *RegistryDirectory) update(res *registry.ServiceEvent) {
	if res == nil {
		return
	}

	log.Debug("registry update, result{%s}", res)

	log.Debug("update service name: %s!", res.Service)

	dir.refreshInvokers(res)
}

func (dir *RegistryDirectory) refreshInvokers(res *registry.ServiceEvent) {
	var newCacheInvokersMap sync.Map

	switch res.Action {
	case registry.ServiceAdd:
		//dir.cacheService.Add(res.Service, dir.serviceTTL)
		newCacheInvokersMap = *dir.cacheInvoker(res.Service)
	case registry.ServiceDel:
		//dir.cacheService.Del(res.Service, dir.serviceTTL)
		newCacheInvokersMap = *dir.uncacheInvoker(res.Service)
		log.Info("selector delete service url{%s}", res.Service)
	default:
		return
	}

	newInvokers := dir.toGroupInvokers(&newCacheInvokersMap)

	dir.listenerLock.Lock()
	defer dir.listenerLock.Unlock()
	dir.cacheInvokers = newInvokers
}

func (dir *RegistryDirectory) toGroupInvokers(newInvokersMap *sync.Map) []protocol.Invoker {

	newInvokersList := []protocol.Invoker{}
	groupInvokersMap := make(map[string][]protocol.Invoker)
	groupInvokersList := []protocol.Invoker{}

	newInvokersMap.Range(func(key, value interface{}) bool {
		newInvokersList = append(newInvokersList, value.(protocol.Invoker))
		return true
	})

	for _, invoker := range newInvokersList {
		group := invoker.GetUrl().GetParam(constant.GROUP_KEY, "")

		if _, ok := groupInvokersMap[group]; ok {
			groupInvokersMap[group] = append(groupInvokersMap[group], invoker)
		} else {
			groupInvokersMap[group] = []protocol.Invoker{}
		}
	}
	if len(groupInvokersMap) == 1 {
		//len is 1 it means no group setting ,so do not need cluster again
		groupInvokersList = groupInvokersMap[""]
	} else {
		for _, invokers := range groupInvokersMap {
			staticDir := directory.NewStaticDirectory(invokers)
			cluster := extension.GetCluster(dir.GetUrl().SubURL.Params.Get(constant.CLUSTER_KEY))
			groupInvokersList = append(groupInvokersList, cluster.Join(staticDir))
		}
	}

	return groupInvokersList
}

func (dir *RegistryDirectory) uncacheInvoker(url config.URL) *sync.Map {
	log.Debug("service will be deleted in cache invokers: invokers key is  %s!", url.String())
	newCacheInvokers := dir.cacheInvokersMap
	newCacheInvokers.Delete(url.String())
	return newCacheInvokers
}

func (dir *RegistryDirectory) cacheInvoker(url config.URL) *sync.Map {
	//check the url's protocol is equal to the protocol which is configured in reference config
	referenceUrl := dir.GetUrl().SubURL
	newCacheInvokers := dir.cacheInvokersMap
	if url.Protocol == referenceUrl.Protocol {
		url = mergeUrl(url, referenceUrl)

		if _, ok := newCacheInvokers.Load(url.String()); !ok {

			log.Debug("service will be added in cache invokers: invokers key is  %s!", url.String())
			newInvoker := extension.GetProtocolExtension(protocolwrapper.FILTER).Refer(url)
			newCacheInvokers.Store(url.String(), newInvoker)
		}
	}
	return newCacheInvokers
}

//select the protocol invokers from the directory
func (dir *RegistryDirectory) List(invocation protocol.Invocation) []protocol.Invoker {
	//TODO:router
	return dir.cacheInvokers
}

func (dir *RegistryDirectory) IsAvailable() bool {
	return true
}

func (dir *RegistryDirectory) Destroy() {
	dir.BaseDirectory.Destroy()
}

// configuration  > reference config >service config
//  in this function we should merge the reference local url config into the service url from registry.
//TODO configuration merge, in the future , the configuration center's config should merge too.
func mergeUrl(serviceUrl config.URL, referenceUrl *config.URL) config.URL {
	mergedUrl := serviceUrl
	var methodConfigMergeFcn = []func(method string){}

	//loadBalance strategy config
	if v := referenceUrl.Params.Get(constant.LOADBALANCE_KEY); v != "" {
		mergedUrl.Params.Set(constant.LOADBALANCE_KEY, v)
	}
	methodConfigMergeFcn = append(methodConfigMergeFcn, func(method string) {
		if v := referenceUrl.Params.Get(method + "." + constant.LOADBALANCE_KEY); v != "" {
			mergedUrl.Params.Set(method+"."+constant.LOADBALANCE_KEY, v)
		}
	})

	//cluster strategy config
	if v := referenceUrl.Params.Get(constant.CLUSTER_KEY); v != "" {
		mergedUrl.Params.Set(constant.CLUSTER_KEY, v)
	}
	methodConfigMergeFcn = append(methodConfigMergeFcn, func(method string) {
		if v := referenceUrl.Params.Get(method + "." + constant.CLUSTER_KEY); v != "" {
			mergedUrl.Params.Set(method+"."+constant.CLUSTER_KEY, v)
		}
	})

	//remote timestamp
	if v := serviceUrl.Params.Get(constant.TIMESTAMP_KEY); v != "" {
		mergedUrl.Params.Set(constant.REMOTE_TIMESTAMP_KEY, v)
		mergedUrl.Params.Set(constant.TIMESTAMP_KEY, referenceUrl.Params.Get(constant.TIMESTAMP_KEY))
	}

	//finally execute methodConfigMergeFcn
	for _, method := range referenceUrl.Methods {
		for _, fcn := range methodConfigMergeFcn {
			fcn("methods." + method)
		}
	}

	return mergedUrl
}
