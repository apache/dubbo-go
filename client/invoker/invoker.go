package invoker

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"
)

import (
	log "github.com/AlexStocks/log4go"
	jerrors "github.com/juju/errors"
)

import (
	"github.com/dubbo/dubbo-go/client"
	"github.com/dubbo/dubbo-go/client/selector"
	"github.com/dubbo/dubbo-go/dubbo"
	"github.com/dubbo/dubbo-go/jsonrpc"
	"github.com/dubbo/dubbo-go/registry"
)

const RegistryConnDelay = 3

type Options struct {
	ServiceTTL time.Duration
	selector   selector.Selector
	// TODO:we should provider a transport client interface
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
	reqIdCounter    int64
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

// Deprecated: use ImplementService instead.
func (ivk *Invoker) DubboCall(reqId int64, registryConf registry.ServiceConfig, method string, args, reply interface{}, opts ...dubbo.CallOption) error {
	return ivk.dubboCall(reqId, registryConf, method, args, reply, opts...)
}

func (ivk *Invoker) dubboCall(reqId int64, registryConf registry.ServiceConfig, method string, args, reply interface{}, opts ...dubbo.CallOption) error {
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
	// TODO:这里要改一下call方法改为接收指针类型
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

func (ivk Invoker) GenReqId() int64 {
	ivk.reqIdCounter++
	return ivk.reqIdCounter
}

var typError = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem()).Type()

func (ivk Invoker) ImplementService(i interface{}, registryConf registry.ServiceConfig, opts ...dubbo.CallOption) error {
	// check parameters, incoming interface must be a elem's pointer.
	valueOf := reflect.ValueOf(i)
	log.Debug("[ImplementService] reflect.TypeOf: %s", valueOf.String())
	if valueOf.Kind() != reflect.Ptr {
		return fmt.Errorf("%s must be a pointer", valueOf)
	}

	valueOfElem := reflect.ValueOf(i).Elem()
	typeOf := valueOfElem.Type()

	// check incoming interface, incoming interface's elem must be a elem.
	switch {
	case valueOfElem.Type().Kind() == reflect.Ptr:
		return fmt.Errorf("%s is a pointer", valueOf)
	case valueOfElem.Type().Kind() != reflect.Struct:
		return fmt.Errorf("%s must be a struct", valueOf) // or interface?
	}

	makeDubboCallProxy := func(methodName string, outs []reflect.Type) func(in []reflect.Value) []reflect.Value {
		return func(in []reflect.Value) []reflect.Value {
			// Convert input parameters to interface.
			var argsInterface = make([]interface{}, len(in))
			for k, v := range in {
				argsInterface[k] = v.Interface()
			}

			var reply reflect.Value
			if outs[0].Kind() == reflect.Ptr {
				reply = reflect.New(outs[0].Elem())
			} else {
				reply = reflect.New(outs[0])
			}

			err := ivk.dubboCall(ivk.GenReqId(), registryConf, methodName, argsInterface, reply.Interface(), opts...)

			if outs[0].Kind() == reflect.Ptr {
			} else {
				reply = reply.Elem()
			}

			// because only one return value in java,
			// so we detect the method we will proxy that return count,
			// if count is 2 then last one must be error.
			switch len(outs) {
			case 1:
				if err != nil {
					panic(err)
				}
				return []reflect.Value{reply}
			case 2:
				return []reflect.Value{reply, reflect.ValueOf(&err).Elem()}
			default:
				panic(fmt.Errorf("%s must returns %s or (%s, error)", methodName, reply.Type().Name(), reply.Type().Name()))
			}
		}
	}

	// enumerate all interface methods or struct filed
	// TODO: add interface support in golang 2.0
	// see more: https://github.com/golang/go/issues/16522
	for i := 0; i < valueOfElem.NumField(); i++ {
		t := typeOf.Field(i)
		f := valueOfElem.Field(i)

		if f.Kind() == reflect.Func && f.IsValid() && f.CanSet() {
			if t.Type.NumOut() < 1 || t.Type.NumOut() > 2 {
				return fmt.Errorf("%s returns must be 1 or 2", t.Type)
			}

			var funcOuts = make([]reflect.Type, t.Type.NumOut())
			for i := 0; i < t.Type.NumOut(); i++ {
				funcOuts[i] = t.Type.Out(i)
			}

			switch t.Type.NumOut() {
			case 2:
				// check second out, must be error.
				if funcOuts[1].Kind() != typError.Kind() {
					return fmt.Errorf("%s returns must be (%s, error)", t.Name, funcOuts[0].Name())
				}
				fallthrough
			default:
				switch {
				case funcOuts[0].Kind() == reflect.Ptr && funcOuts[0].Elem().Kind() == reflect.Ptr:
					return fmt.Errorf("%s returns[1] type %s's elem still is a pointer", t.Name, funcOuts[0].Name())
				}
			}

			// do method proxy here:
			f.Set(reflect.MakeFunc(f.Type(), makeDubboCallProxy(t.Name, funcOuts)))
		}
	}

	return nil
}
