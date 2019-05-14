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
	"github.com/dubbo/go-for-apache-dubbo/cluster/directory"
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	"github.com/dubbo/go-for-apache-dubbo/protocol/protocolwrapper"
	"github.com/dubbo/go-for-apache-dubbo/registry"
)

const RegistryConnDelay = 3

type Options struct {
	serviceTTL time.Duration
}
type Option func(*Options)

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

func NewRegistryDirectory(url *common.URL, registry registry.Registry, opts ...Option) (*RegistryDirectory, error) {
	options := Options{
		//default 300s
		serviceTTL: time.Duration(300e9),
	}
	for _, opt := range opts {
		opt(&options)
	}
	if url.SubURL == nil {
		return nil, jerrors.Errorf("url is invalid, suburl can not be nil")
	}
	return &RegistryDirectory{
		BaseDirectory:    directory.NewBaseDirectory(url),
		cacheInvokers:    []protocol.Invoker{},
		cacheInvokersMap: &sync.Map{},
		serviceType:      url.SubURL.Service(),
		registry:         registry,
		Options:          options,
	}, nil
}

//subscibe from registry
func (dir *RegistryDirectory) Subscribe(url common.URL) {
	for {
		if !dir.registry.IsAvailable() {
			log.Warn("event listener game over.")
			return
		}

		listener, err := dir.registry.Subscribe(url)
		if err != nil {
			if !dir.registry.IsAvailable() {
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
				log.Info("update begin, service event: %v", serviceEvent.String())
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
		//dir.cacheService.Add(res.Path, dir.serviceTTL)
		newCacheInvokersMap = *dir.cacheInvoker(res.Service)
	case registry.ServiceDel:
		//dir.cacheService.Del(res.Path, dir.serviceTTL)
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
			groupInvokersMap[group] = []protocol.Invoker{invoker}
		}
	}
	if len(groupInvokersMap) == 1 {
		//len is 1 it means no group setting ,so do not need cluster again
		groupInvokersList = groupInvokersMap[""]
	} else {
		for _, invokers := range groupInvokersMap {
			staticDir := directory.NewStaticDirectory(invokers)
			cluster := extension.GetCluster(dir.GetUrl().SubURL.GetParam(constant.CLUSTER_KEY, constant.DEFAULT_CLUSTER))
			groupInvokersList = append(groupInvokersList, cluster.Join(staticDir))
		}
	}

	return groupInvokersList
}

func (dir *RegistryDirectory) uncacheInvoker(url common.URL) *sync.Map {
	log.Debug("service will be deleted in cache invokers: invokers key is  %s!", url.Key())
	newCacheInvokers := dir.cacheInvokersMap
	newCacheInvokers.Delete(url.Key())
	return newCacheInvokers
}

func (dir *RegistryDirectory) cacheInvoker(url common.URL) *sync.Map {
	referenceUrl := dir.GetUrl().SubURL
	newCacheInvokers := dir.cacheInvokersMap
	//check the url's protocol is equal to the protocol which is configured in reference config or referenceUrl is not care about protocol
	if url.Protocol == referenceUrl.Protocol || referenceUrl.Protocol == "" {
		url = mergeUrl(url, referenceUrl)

		if _, ok := newCacheInvokers.Load(url.Key()); !ok {
			log.Debug("service will be added in cache invokers: invokers key is  %s!", url.Key())
			newInvoker := extension.GetProtocolExtension(protocolwrapper.FILTER).Refer(url)
			if newInvoker != nil {
				newCacheInvokers.Store(url.Key(), newInvoker)
			}
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
	return dir.BaseDirectory.IsAvailable()
}

func (dir *RegistryDirectory) Destroy() {
	//TODO:unregister & unsubscribe
	dir.BaseDirectory.Destroy(func() {
		for _, ivk := range dir.cacheInvokers {
			ivk.Destroy()
		}
		dir.cacheInvokers = []protocol.Invoker{}
	})
}

// configuration  > reference config >service config
//  in this function we should merge the reference local url config into the service url from registry.
//TODO configuration merge, in the future , the configuration center's config should merge too.
func mergeUrl(serviceUrl common.URL, referenceUrl *common.URL) common.URL {
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
