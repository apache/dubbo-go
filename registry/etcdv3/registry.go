package etcdv3

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/common/utils"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting/etcdv3"
)

var (
	processID = ""
	localIP   = ""
)

const (
	Name              = "etcdv3"
	RegistryConnDelay = 3
)

func init() {
	processID = fmt.Sprintf("%d", os.Getpid())
	localIP, _ = utils.GetLocalIP()
	extension.SetRegistry(Name, newETCDV3Registry)
}

type etcdV3Registry struct {
	*common.URL
	birth int64 // time of file birth, seconds since Epoch; 0 if unknown

	cltLock  sync.Mutex
	client   *etcdv3.Client
	services map[string]common.URL // service name + protocol -> service config

	listenerLock   sync.Mutex
	listener       *etcdv3.EventListener
	dataListener   *dataListener
	configListener *configurationListener

	wg   sync.WaitGroup // wg+done for etcd client restart
	done chan struct{}
}

func (r *etcdV3Registry) Client() *etcdv3.Client {
	return r.client
}
func (r *etcdV3Registry) SetClient(client *etcdv3.Client) {
	r.client = client
}
func (r *etcdV3Registry) ClientLock() *sync.Mutex {
	return &r.cltLock
}
func (r *etcdV3Registry) WaitGroup() *sync.WaitGroup {
	return &r.wg
}
func (r *etcdV3Registry) GetDone() chan struct{} {
	return r.done
}
func (r *etcdV3Registry) RestartCallBack() bool {

	services := []common.URL{}
	for _, confIf := range r.services {
		services = append(services, confIf)
	}

	flag := true
	for _, confIf := range services {
		err := r.Register(confIf)
		if err != nil {
			logger.Errorf("(etcdV3ProviderRegistry)register(conf{%#v}) = error{%#v}",
				confIf, perrors.WithStack(err))
			flag = false
			break
		}
		logger.Infof("success to re-register service :%v", confIf.Key())
	}
	return flag
}

func newETCDV3Registry(url *common.URL) (registry.Registry, error) {

	timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
	if err != nil {
		logger.Errorf("timeout config %v is invalid ,err is %v",
			url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT), err.Error())
		return nil, perrors.WithMessagef(err, "new etcd registry(address:%+v)", url.Location)
	}

	logger.Infof("etcd address is: %v, timeout is: %s", url.Location, timeout.String())

	r := &etcdV3Registry{
		URL:      url,
		birth:    time.Now().UnixNano(),
		done:     make(chan struct{}),
		services: make(map[string]common.URL),
	}

	if err := etcdv3.ValidateClient(
		r,
		etcdv3.WithName(etcdv3.RegistryETCDV3Client),
		etcdv3.WithTimeout(timeout),
		etcdv3.WithEndpoints(url.Location),
	); err != nil {
		return nil, err
	}

	r.wg.Add(1)
	go etcdv3.HandleClientRestart(r)

	r.listener = etcdv3.NewEventListener(r.client)
	r.configListener = NewConfigurationListener(r)
	r.dataListener = NewRegistryDataListener(r.configListener)

	return r, nil
}

func (r *etcdV3Registry) GetUrl() common.URL {
	return *r.URL
}

func (r *etcdV3Registry) IsAvailable() bool {

	select {
	case <-r.done:
		return false
	default:
		return true
	}
}

func (r *etcdV3Registry) Destroy() {

	if r.configListener != nil {
		r.configListener.Close()
	}
	r.stop()
}

func (r *etcdV3Registry) stop() {

	close(r.done)

	// close current client
	r.client.Close()

	r.cltLock.Lock()
	r.client = nil
	r.services = nil
	r.cltLock.Unlock()
}

func (r *etcdV3Registry) Register(svc common.URL) error {

	role, err := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	if err != nil {
		return perrors.WithMessage(err, "get registry role")
	}

	r.cltLock.Lock()
	if _, ok := r.services[svc.Key()]; ok {
		r.cltLock.Unlock()
		return perrors.New(fmt.Sprintf("Path{%s} has been registered", svc.Path))
	}
	r.cltLock.Unlock()

	switch role {
	case common.PROVIDER:
		logger.Debugf("(provider register )Register(conf{%#v})", svc)
		if err := r.registerProvider(svc); err != nil {
			return perrors.WithMessage(err, "register provider")
		}
	case common.CONSUMER:
		logger.Debugf("(consumer register )Register(conf{%#v})", svc)
		if err := r.registerConsumer(svc); err != nil {
			return perrors.WithMessage(err, "register consumer")
		}
	default:
		return perrors.New(fmt.Sprintf("unknown role %d", role))
	}

	r.cltLock.Lock()
	r.services[svc.Key()] = svc
	r.cltLock.Unlock()
	return nil
}

func (r *etcdV3Registry) createDirIfNotExist(k string) error {

	var tmpPath string
	for _, str := range strings.Split(k, "/")[1:] {
		tmpPath = path.Join(tmpPath, "/", str)
		if err := r.client.Create(tmpPath, ""); err != nil {
			return perrors.WithMessagef(err, "create path %s in etcd", tmpPath)
		}
	}

	return nil
}

func (r *etcdV3Registry) registerConsumer(svc common.URL) error {

	consumersNode := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), common.DubboNodes[common.CONSUMER])
	if err := r.createDirIfNotExist(consumersNode); err != nil {
		logger.Errorf("etcd client create path %s: %v", consumersNode, err)
		return perrors.WithMessage(err, "etcd create consumer nodes")
	}
	providersNode := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), common.DubboNodes[common.PROVIDER])
	if err := r.createDirIfNotExist(providersNode); err != nil {
		return perrors.WithMessage(err, "create provider node")
	}

	params := url.Values{}

	params.Add("protocol", svc.Protocol)

	params.Add("category", (common.RoleType(common.CONSUMER)).String())
	params.Add("dubbo", "dubbogo-consumer-"+constant.Version)

	encodedURL := url.QueryEscape(fmt.Sprintf("consumer://%s%s?%s", localIP, svc.Path, params.Encode()))
	dubboPath := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), (common.RoleType(common.CONSUMER)).String())
	if err := r.client.Create(path.Join(dubboPath, encodedURL), ""); err != nil {
		return perrors.WithMessagef(err, "create k/v in etcd (path:%s, url:%s)", dubboPath, encodedURL)
	}

	return nil
}

func (r *etcdV3Registry) registerProvider(svc common.URL) error {

	if len(svc.Path) == 0 || len(svc.Methods) == 0 {
		return perrors.New(fmt.Sprintf("service path %s or service method %s", svc.Path, svc.Methods))
	}

	var (
		urlPath    string
		encodedURL string
		dubboPath  string
	)

	providersNode := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), common.DubboNodes[common.PROVIDER])
	if err := r.createDirIfNotExist(providersNode); err != nil {
		return perrors.WithMessage(err, "create provider node")
	}

	params := url.Values{}

	svc.RangeParams(func(key, value string) bool {
		params[key] = []string{value}
		return true
	})
	params.Add("pid", processID)
	params.Add("ip", localIP)
	params.Add("anyhost", "true")
	params.Add("category", (common.RoleType(common.PROVIDER)).String())
	params.Add("dubbo", "dubbo-provider-golang-"+constant.Version)
	params.Add("side", (common.RoleType(common.PROVIDER)).Role())

	if len(svc.Methods) == 0 {
		params.Add("methods", strings.Join(svc.Methods, ","))
	}

	logger.Debugf("provider url params:%#v", params)
	var host string
	if len(svc.Ip) == 0 {
		host = localIP + ":" + svc.Port
	} else {
		host = svc.Ip + ":" + svc.Port
	}

	urlPath = svc.Path

	encodedURL = url.QueryEscape(fmt.Sprintf("%s://%s%s?%s", svc.Protocol, host, urlPath, params.Encode()))
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", svc.Service(), (common.RoleType(common.PROVIDER)).String())

	if err := r.client.Create(path.Join(dubboPath, encodedURL), ""); err != nil {
		return perrors.WithMessagef(err, "create k/v in etcd (path:%s, url:%s)", dubboPath, encodedURL)
	}

	return nil
}

func (r *etcdV3Registry) subscribe(svc *common.URL) (registry.Listener, error) {

	var (
		configListener *configurationListener
	)

	r.listenerLock.Lock()
	configListener = r.configListener
	r.listenerLock.Unlock()
	if r.listener == nil {
		r.cltLock.Lock()
		client := r.client
		r.cltLock.Unlock()
		if client == nil {
			return nil, perrors.New("etcd client broken")
		}

		// new client & listener
		listener := etcdv3.NewEventListener(r.client)

		r.listenerLock.Lock()
		r.listener = listener
		r.listenerLock.Unlock()
	}

	//register the svc to dataListener
	r.dataListener.AddInterestedURL(svc)
	for _, v := range strings.Split(svc.GetParam(constant.CATEGORY_KEY, constant.DEFAULT_CATEGORY), ",") {
		go r.listener.ListenServiceEvent(fmt.Sprintf("/dubbo/%s/"+v, svc.Service()), r.dataListener)
	}

	return configListener, nil
}

//subscibe from registry
func (r *etcdV3Registry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) {
	for {
		if !r.IsAvailable() {
			logger.Warnf("event listener game over.")
			return
		}

		listener, err := r.subscribe(url)
		if err != nil {
			if !r.IsAvailable() {
				logger.Warnf("event listener game over.")
				return
			}
			logger.Warnf("getListener() = err:%v", perrors.WithStack(err))
			time.Sleep(time.Duration(RegistryConnDelay) * time.Second)
			continue
		}

		for {
			if serviceEvent, err := listener.Next(); err != nil {
				logger.Warnf("Selector.watch() = error{%v}", perrors.WithStack(err))
				listener.Close()
				return
			} else {
				logger.Infof("update begin, service event: %v", serviceEvent.String())
				notifyListener.Notify(serviceEvent)
			}

		}

	}
}
