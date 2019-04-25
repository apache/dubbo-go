package invoker

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
	"github.com/dubbo/go-for-apache-dubbo/client"
	"github.com/dubbo/go-for-apache-dubbo/client/selector"
	"github.com/dubbo/go-for-apache-dubbo/dubbo"
	"github.com/dubbo/go-for-apache-dubbo/jsonrpc"
	"github.com/dubbo/go-for-apache-dubbo/registry"
)

const RegistryConnDelay = 3

type Options struct {
	ServiceTTL time.Duration
	selector   selector.Selector
	//TODO:we should provider a transport client interface
	HttpClient  *jsonrpc.HTTPClient
	DubboClient *dubbo.Client
}
type Option func(*Options)

func WithServiceTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.ServiceTTL = ttl
	}
}

func WithHttpClient(client *jsonrpc.HTTPClient) Option {
	return func(o *Options) {
		o.HttpClient = client
	}
}
func WithDubboClient(client *dubbo.Client) Option {
	return func(o *Options) {
		o.DubboClient = client
	}
}

func WithLBSelector(selector selector.Selector) Option {
	return func(o *Options) {
		o.selector = selector
	}
}

type Invoker struct {
	Options
	cacheServiceMap map[string]*ServiceArray
	registry        registry.Registry
	listenerLock    sync.Mutex
}

func NewInvoker(registry registry.Registry, opts ...Option) (*Invoker, error) {
	options := Options{
		//default 300s
		ServiceTTL: time.Duration(300e9),
		selector:   selector.NewRandomSelector(),
	}
	for _, opt := range opts {
		opt(&options)
	}
	if options.HttpClient == nil && options.DubboClient == nil {
		return nil, jerrors.New("Must specify the transport client!")
	}
	invoker := &Invoker{
		Options:         options,
		cacheServiceMap: make(map[string]*ServiceArray),
		registry:        registry,
	}
	go invoker.listen()
	return invoker, nil
}

func (ivk *Invoker) listen() {
	for {
		if ivk.registry.IsClosed() {
			log.Warn("event listener game over.")
			return
		}

		listener, err := ivk.registry.Subscribe()
		if err != nil {
			if ivk.registry.IsClosed() {
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
				ivk.update(serviceEvent)
			}

		}

	}
}

func (ivk *Invoker) update(res *registry.ServiceEvent) {
	if res == nil || res.Service == nil {
		return
	}

	log.Debug("registry update, result{%s}", res)
	registryKey := res.Service.ServiceConfig().Key()

	ivk.listenerLock.Lock()
	defer ivk.listenerLock.Unlock()

	svcArr, ok := ivk.cacheServiceMap[registryKey]
	log.Debug("registry name:%s, its current member lists:%+v", registryKey, svcArr)

	switch res.Action {
	case registry.ServiceAdd:
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

func (ivk *Invoker) getService(registryConf registry.ServiceConfig) (*ServiceArray, error) {
	defer ivk.listenerLock.Unlock()

	registryKey := registryConf.Key()

	ivk.listenerLock.Lock()
	svcArr, sok := ivk.cacheServiceMap[registryKey]
	log.Debug("r.svcArr[registryString{%v}] = svcArr{%s}", registryKey, svcArr)
	if sok && time.Since(svcArr.birth) < ivk.Options.ServiceTTL {
		return svcArr, nil
	}
	ivk.listenerLock.Unlock()

	svcs, err := ivk.registry.GetService(registryConf)
	ivk.listenerLock.Lock()

	if err != nil {
		log.Error("Registry.get(conf:%+v) = {err:%s, svcs:%+v}",
			registryConf, jerrors.ErrorStack(err), svcs)

		return nil, jerrors.Trace(err)
	}

	newSvcArr := newServiceArray(svcs)
	ivk.cacheServiceMap[registryKey] = newSvcArr
	return newSvcArr, nil
}

func (ivk *Invoker) HttpCall(ctx context.Context, reqId int64, req client.Request, resp interface{}) error {

	serviceConf := req.ServiceConfig()
	registryArray, err := ivk.getService(serviceConf)
	if err != nil {
		return err
	}
	if len(registryArray.arr) == 0 {
		return jerrors.New("cannot find svc " + serviceConf.String())
	}
	url, err := ivk.selector.Select(reqId, registryArray)
	if err != nil {
		return err
	}
	if err = ivk.HttpClient.Call(ctx, url, req, resp); err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		return err
	}
	log.Info("response result:%s", resp)
	return nil
}

func (ivk *Invoker) DubboCall(reqId int64, registryConf registry.ServiceConfig, method string, args, reply interface{}, opts ...dubbo.CallOption) error {

	registryArray, err := ivk.getService(registryConf)
	if err != nil {
		return err
	}
	if len(registryArray.arr) == 0 {
		return jerrors.New("cannot find svc " + registryConf.String())
	}
	url, err := ivk.selector.Select(reqId, registryArray)
	if err != nil {
		return err
	}
	//TODO:这里要改一下call方法改为接收指针类型
	if err = ivk.DubboClient.Call(url.Ip()+":"+url.Port(), url, method, args, reply, opts...); err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		return err
	}
	log.Info("response result:%s", reply)
	return nil
}

func (ivk *Invoker) Close() {
	ivk.DubboClient.Close()
}
