package etcd

import (
	"context"
	"fmt"
	"github.com/apache/dubbo-go/remoting"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	etcd "github.com/AlexStocks/goext/database/etcd"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/common/utils"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/version"
	"github.com/juju/errors"
	"go.etcd.io/etcd/clientv3"
)

var (
	processID = ""
	localIP   = ""
)

func init() {
	processID = fmt.Sprintf("%d", os.Getpid())
	localIP, _ = utils.GetLocalIP()
	extension.SetRegistry("etcd", newETCDV3Registry)
}

type etcdV3Registry struct {
	*common.URL
	birth int64 // time of file birth, seconds since Epoch; 0 if unknown

	ctx    context.Context
	cancel context.CancelFunc

	rawClient *clientv3.Client
	client    *etcd.Client

	dataListener   remoting.DataListener
	configListener remoting.ConfigurationListener

	servicesCache sync.Map // service name + protocol -> service config
}

func newETCDV3Registry(url *common.URL) (registry.Registry, error) {

	timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
	if err != nil {
		logger.Errorf("timeout config %v is invalid ,err is %v",
			url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT), err.Error())
		return nil, errors.Annotatef(err, "new etcd registry(address:%+v)", url.Location)
	}

	logger.Infof("etcd address is: %v", url.Location)
	logger.Infof("time-out is: %v", timeout.String())

	rawClient, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{url.Location},
		DialTimeout: timeout,
		//DialOptions: []grpc.DialOption{grpc.WithBlock()},
	})
	if err != nil {
		return nil, errors.Annotate(err, "block connect to etcd server")
	}

	rootCtx, cancel := context.WithCancel(context.Background())
	client, err := etcd.NewClient(rawClient, etcd.WithTTL(time.Second), etcd.WithContext(rootCtx))
	if err != nil {
		return nil, errors.Annotate(err, "new etcd client")
	}

	r := etcdV3Registry{
		URL:           url,
		ctx:           rootCtx,
		cancel:        cancel,
		rawClient:     rawClient,
		client:        client,
		servicesCache: sync.Map{},
	}

	go r.keepAlive()
	return &r, nil
}

func (r *etcdV3Registry) keepAlive() error {

	resp, err := r.client.KeepAlive()
	if err != nil {
		return errors.Annotate(err, "keep alive")
	}
	go func() {
		for {
			select {
			case _, ok := <-resp:
				if !ok {
					logger.Errorf("etcd server stop")
					r.cancel()
					return
				}

			}
		}
	}()
	return nil
}

func (r *etcdV3Registry) GetUrl() common.URL {
	return *r.URL
}

func (r *etcdV3Registry) IsAvailable() bool {

	select {
	case <-r.ctx.Done():
		return false
	default:
		return true
	}
}

func (r *etcdV3Registry) Destroy() {
	r.stop()
}

func (r *etcdV3Registry) stop() {

	// close current client
	r.rawClient.Close()

	// cancel ctx
	r.cancel()

	r.rawClient = nil
	r.ctx = nil
	r.cancel = nil
	r.servicesCache.Range(func(key, value interface{}) bool {
		r.servicesCache.Delete(key)
		return true
	}) // empty service catalog
}

func (r *etcdV3Registry) Register(svc common.URL) error {

	role, err := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	if err != nil {
		return errors.Annotate(err, "get registry role")
	}

	if _, ok := r.servicesCache.Load(svc.Key()); ok {
		return errors.New(fmt.Sprintf("Path{%s} has been registered", svc.Path))
	}

	switch role {
	case common.PROVIDER:
		logger.Debugf("(provider register )Register(conf{%#v})", svc)
		if err := r.registerProvider(svc); err != nil {
			return errors.Annotate(err, "register provider")
		}
	case common.CONSUMER:
		logger.Debugf("(consumer register )Register(conf{%#v})", svc)
		if err := r.registerConsumer(svc); err != nil {
			return errors.Annotate(err, "register consumer")
		}
	default:
		return errors.New(fmt.Sprintf("unknown role %d", role))
	}

	r.servicesCache.Store(svc.Key(), svc)
	return nil
}

func (r *etcdV3Registry) createKVIfNotExist(k string, v string) error {

	_, err := r.rawClient.Txn(r.ctx).
		If(clientv3.Compare(clientv3.Version(k), "<", 1)).
		Then(clientv3.OpPut(k, v)).
		Commit()
	if err != nil {
		return errors.Annotatef(err, "etcd create k %s v %s", k, v)
	}
	return nil
}

func (r *etcdV3Registry) createDirIfNotExist(k string) error {

	var tmpPath string
	for _, str := range strings.Split(k, "/")[1:] {
		tmpPath = path.Join(tmpPath, "/", str)
		if err := r.createKVIfNotExist(tmpPath, ""); err != nil {
			return errors.Annotatef(err, "create path %s in etcd", tmpPath)
		}
	}

	return nil
}

func (r *etcdV3Registry) registerConsumer(svc common.URL) error {

	consumersNode := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), common.DubboNodes[common.CONSUMER])
	if err := r.createDirIfNotExist(consumersNode); err != nil {
		logger.Errorf("etcd client create path %s: %v", consumersNode, err)
		return errors.Annotate(err, "etcd create consumer nodes")
	}
	providersNode := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), common.DubboNodes[common.PROVIDER])
	if err := r.createDirIfNotExist(providersNode); err != nil {
		return errors.Annotate(err, "create provider node")
	}

	params := url.Values{}

	params.Add("protocol", svc.Protocol)

	params.Add("category", (common.RoleType(common.CONSUMER)).String())
	params.Add("dubbo", "dubbogo-consumer-"+version.Version)

	encodedURL := url.QueryEscape(fmt.Sprintf("consumer://%s%s?%s", localIP, svc.Path, params.Encode()))
	dubboPath := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), (common.RoleType(common.CONSUMER)).String())
	if err := r.createKVIfNotExist(path.Join(dubboPath, encodedURL), ""); err != nil {
		return errors.Annotatef(err, "create k/v in etcd (path:%s, url:%s)", dubboPath, encodedURL)
	}

	return nil
}

func (r *etcdV3Registry) registerProvider(svc common.URL) error {

	if svc.Path == "" || len(svc.Methods) == 0 {
		return errors.New(fmt.Sprintf("service path %s or service method %s", svc.Path, svc.Methods))
	}

	var (
		urlPath    string
		encodedURL string
		dubboPath  string
	)

	providersNode := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), common.DubboNodes[common.PROVIDER])
	if err := r.createDirIfNotExist(providersNode); err != nil {
		return errors.Annotate(err, "create provider node")
	}

	params := url.Values{}
	for k, v := range svc.Params {
		params[k] = v
	}

	params.Add("pid", processID)
	params.Add("ip", localIP)
	params.Add("anyhost", "true")
	params.Add("category", (common.RoleType(common.PROVIDER)).String())
	params.Add("dubbo", "dubbo-provider-golang-"+version.Version)
	params.Add("side", (common.RoleType(common.PROVIDER)).Role())

	if len(svc.Methods) == 0 {
		params.Add("methods", strings.Join(svc.Methods, ","))
	}

	logger.Debugf("provider url params:%#v", params)
	var host string
	if svc.Ip == "" {
		host = localIP + ":" + svc.Port
	} else {
		host = svc.Ip + ":" + svc.Port
	}

	urlPath = svc.Path

	encodedURL = url.QueryEscape(fmt.Sprintf("%s://%s%s?%s", svc.Protocol, host, urlPath, params.Encode()))
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", svc.Service(), (common.RoleType(common.PROVIDER)).String())

	if err := r.createKVIfNotExist(path.Join(dubboPath, encodedURL), ""); err != nil {
		return errors.Annotatef(err, "create k/v in etcd (path:%s, url:%s)", dubboPath, encodedURL)
	}

	return nil
}

func (r *etcdV3Registry) Subscribe(svc common.URL) (registry.Listener, error) {


	logger.Infof("subscribe svc: %s", svc)

	return nil, nil
}
