package invoker

import (
	"context"
	"github.com/dubbo/dubbo-go/client"
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/client/loadBalance"
	"github.com/dubbo/dubbo-go/registry"
	"github.com/dubbo/dubbo-go/service"
)

type Options struct {
	ServiceTTL time.Duration
	selector   loadBalance.Selector
	Transport  client.Transport
}
type Option func(*Options)

func WithServiceTTL(ttl time.Duration) Option {
	return func(o *Options) {
		o.ServiceTTL = ttl
	}
}

func WithClientTransport(client client.Transport) Option {
	return func(o *Options) {
		o.Transport = client
	}
}

func WithLBSelector(selector loadBalance.Selector) Option {
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
		selector:   loadBalance.NewRandomSelector(),
	}
	for _, opt := range opts {
		opt(&options)
	}
	if options.Transport == nil {
		return nil, jerrors.New("Must specify the client transport !")
	}
	invoker := &Invoker{
		Options:         options,
		cacheServiceMap: make(map[string]*ServiceArray),
		registry:        registry,
	}
	invoker.Listen()
	return invoker, nil
}

func (ivk *Invoker) Listen() {
	go ivk.listen()
}

func (ivk *Invoker) listen() {
	for {
		ch := ivk.registry.GetListenEvent()

		for {
			e, isOpen := <-ch
			if !isOpen {
				log.Warn("registry closed!")
				break
			}
			ivk.update(e)
		}

	}
}

func (ivk *Invoker) update(res *registry.ServiceURLEvent) {
	if res == nil || res.Service == nil {
		return
	}

	log.Debug("registry update, result{%s}", res)
	serviceKey := res.Service.ServiceConfig().Key()

	ivk.listenerLock.Lock()
	defer ivk.listenerLock.Unlock()

	svcArr, ok := ivk.cacheServiceMap[serviceKey]
	log.Debug("service name:%s, its current member lists:%+v", serviceKey, svcArr)

	switch res.Action {
	case registry.ServiceURLAdd:
		if ok {
			svcArr.add(res.Service, ivk.ServiceTTL)
		} else {
			ivk.cacheServiceMap[serviceKey] = newServiceArray([]*service.ServiceURL{res.Service})
		}
	case registry.ServiceURLDel:
		if ok {
			svcArr.del(res.Service, ivk.ServiceTTL)
			if len(svcArr.arr) == 0 {
				delete(ivk.cacheServiceMap, serviceKey)
				log.Warn("delete service %s from service map", serviceKey)
			}
		}
		log.Error("selector delete serviceURL{%s}", *res.Service)
	}
}

func (ivk *Invoker) getService(serviceConf *service.ServiceConfig) (*ServiceArray, error) {
	defer ivk.listenerLock.Unlock()

	serviceKey := serviceConf.Key()

	ivk.listenerLock.Lock()
	svcArr, sok := ivk.cacheServiceMap[serviceKey]
	log.Debug("r.svcArr[serviceString{%v}] = svcArr{%s}", serviceKey, svcArr)
	if sok && time.Since(svcArr.birth) < ivk.Options.ServiceTTL {
		return svcArr, nil
	}
	ivk.listenerLock.Unlock()

	svcs, err := ivk.registry.GetService(serviceConf)
	ivk.listenerLock.Lock()

	if err != nil {
		log.Error("Registry.get(conf:%+v) = {err:%s, svcs:%+v}",
			serviceConf, jerrors.ErrorStack(err), svcs)

		return nil, jerrors.Trace(err)
	}

	newSvcArr := newServiceArray(svcs)
	ivk.cacheServiceMap[serviceKey] = newSvcArr
	return newSvcArr, nil
}

func (ivk *Invoker) Call(ctx context.Context, reqId int64, serviceConf *service.ServiceConfig, req client.Request, resp interface{}) error {

	serviceArray, err := ivk.getService(serviceConf)
	if err != nil {
		return err
	}
	if len(serviceArray.arr) == 0 {
		return jerrors.New("cannot find svc " + serviceConf.String())
	}
	url, err := ivk.selector.Select(reqId, serviceArray)
	if err != nil {
		return err
	}
	if err = ivk.Transport.Call(ctx, url, req, resp); err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		return err
	}
	log.Info("response result:%s", resp)
	return nil
}
