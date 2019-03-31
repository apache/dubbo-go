package invoker

import (
	"context"
	"github.com/dubbo/dubbo-go/dubbo"
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/client/selector"
	"github.com/dubbo/dubbo-go/jsonrpc"
	"github.com/dubbo/dubbo-go/registry"
	"github.com/dubbo/dubbo-go/service"
)

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

func (ivk *Invoker) HttpCall(ctx context.Context, reqId int64, serviceConf *service.ServiceConfig, req jsonrpc.Request, resp interface{}) error {

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
	if err = ivk.HttpClient.Call(ctx, url, req, resp); err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		return err
	}
	log.Info("response result:%s", resp)
	return nil
}

func (ivk *Invoker) DubboCall(reqId int64, serviceConf *service.ServiceConfig, method string, args, reply interface{}, opts ...dubbo.CallOption) error {

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
	//TODO:这里要改一下call方法改为接收指针类型
	if err = ivk.DubboClient.Call(url.Ip+":"+url.Port, *url, method, args, reply, opts...); err != nil {
		log.Error("client.Call() return error:%+v", jerrors.ErrorStack(err))
		return err
	}
	log.Info("response result:%s", reply)
	return nil
}
