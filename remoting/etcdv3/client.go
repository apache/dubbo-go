package etcdv3

import (
	"context"
	"path"
	"sync"
	"time"
)

import (
	"github.com/AlexStocks/goext/database/etcd"
	"github.com/juju/errors"
	perrors "github.com/pkg/errors"
	"go.etcd.io/etcd/clientv3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
)

import (
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

const (
	ConnDelay    = 3
	MaxFailTimes = 15
)

var (
	ErrNilETCDV3ClientConn = errors.New("etcdv3client{conn} is nil")
	ErrKVPairNotFound      = errors.New("k/v pair not found")
)

// clientSet for etcdv3
type clientSet struct {
	lock sync.RWMutex // protect all element in

	// clientSet
	gxClient  *gxetcd.Client
	rawClient *clientv3.Client

	// client controller used to change client behave
	ctx    context.Context // if etcd connection lose, the ctx.Done will be sent msg
	cancel context.CancelFunc

	// c was filled, start maintenanceStatus
	startMaintenanceChan chan struct{}

	c *Client
}

func newClientSet(endpoints []string, timeout time.Duration, c *Client) error {

	rootCtx, cancel := context.WithCancel(context.Background())

	// connect to etcd
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: timeout,
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	})
	if err != nil {
		return errors.Annotate(err, "block connect to etcd server")
	}

	// share context
	gxClient, err := gxetcd.NewClient(client, gxetcd.WithTTL(time.Second), gxetcd.WithContext(rootCtx))
	if err != nil {
		return errors.Annotate(err, "new etcd client")
	}

	out := &clientSet{
		gxClient:             gxClient,
		rawClient:            client,
		ctx:                  rootCtx,
		cancel:               cancel,
		startMaintenanceChan: make(chan struct{}),
		c:                    c,
	}

	err = out.maintenanceStatus()
	if err != nil {
		return errors.Annotate(err, "maintenance connection status")
	}

	// set clientset to client
	c.cs = out

	return nil
}

func (c *clientSet) maintenanceStatus() error {

	c.c.Wait.Add(1)
	aliveResp, err := c.gxClient.KeepAlive()
	if err != nil {
		return errors.Annotatef(err, "etcd keep alive")
	}
	// start maintenance the connection status
	go c.maintenanceStatusLoop(aliveResp)
	return nil
}

func (c *clientSet) maintenanceStatusLoop(aliveResp <-chan *clientv3.LeaseKeepAliveResponse) {

	defer func() {
		c.c.Wait.Done()
		logger.Infof("etcd {path:%v, name:%s} connection goroutine game over.", c.c.endpoints, c.c.name)
	}()

	// get signal, will start maintenanceStatusLoop
	<-c.startMaintenanceChan

	for {
		select {
		case <-c.c.Done():
			// client done
			return
		case <-c.ctx.Done():
			// client context exit
			logger.Warn("etcd clientSet context done")
			return
		case msg, ok := <-aliveResp:
			// etcd connection lose
			// NOTICE
			// if clientSet.Client is nil, it will panic
			if !ok {

				logger.Warnf("etcd server stop at term: %#v", msg)

				c.c.Lock() // hold the c.Client lock
				c.c.cs.clean()

				// NOTICE
				// uninstall the cs from client
				c.c.cs = nil
				c.c.Unlock()
				return
			}
		}
	}
}

func (c *clientSet) put(k string, v string, opts ...clientv3.OpOption) error {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return ErrNilETCDV3ClientConn
	}

	_, err := c.rawClient.Txn(c.ctx).
		If(clientv3.Compare(clientv3.Version(k), "<", 1)).
		Then(clientv3.OpPut(k, v, opts...)).
		Commit()
	if err != nil {
		return err

	}
	return nil
}

func (c *clientSet) delete(k string) error {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return ErrNilETCDV3ClientConn
	}

	_, err := c.rawClient.Delete(c.ctx, k)
	if err != nil {
		return err

	}
	return nil
}

func (c *clientSet) get(k string) (string, error) {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return "", ErrNilETCDV3ClientConn
	}

	resp, err := c.rawClient.Get(c.ctx, k)
	if err != nil {
		return "", err
	}

	if len(resp.Kvs) == 0 {
		return "", ErrKVPairNotFound
	}

	return string(resp.Kvs[0].Value), nil
}

func (c *clientSet) getChildrenW(k string) ([]string, []string, clientv3.WatchChan, error) {

	kList, vList, err := c.getChildren(k)
	if err != nil {
		return nil, nil, nil, errors.Annotatef(err, "get children %s", k)
	}

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return nil, nil, nil, ErrNilETCDV3ClientConn
	}

	wc := c.rawClient.Watch(c.ctx, k, clientv3.WithPrefix())
	return kList, vList, wc, nil
}

func (c *clientSet) watch(k string) (clientv3.WatchChan, error) {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return nil, ErrNilETCDV3ClientConn
	}

	_, err := c.get(k)
	if err != nil {
		return nil, errors.Annotatef(err, "watch pre check key %s", k)
	}

	return c.rawClient.Watch(c.ctx, k), nil
}

func (c *clientSet) getChildren(k string) ([]string, []string, error) {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return nil, nil, ErrNilETCDV3ClientConn
	}

	resp, err := c.rawClient.Get(c.ctx, k, clientv3.WithPrefix())
	if err != nil {
		return nil, nil, err
	}

	if len(resp.Kvs) == 0 {
		return nil, nil, ErrKVPairNotFound
	}

	var (
		kList []string
		vList []string
	)

	for _, kv := range resp.Kvs {
		kList = append(kList, string(kv.Key))
		vList = append(vList, string(kv.Value))
	}

	return kList, vList, nil
}

func (c *clientSet) keepAliveKV(k string, v string) error {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return ErrNilETCDV3ClientConn
	}

	lease, err := c.rawClient.Grant(c.ctx, int64(time.Second.Seconds()))
	if err != nil {
		return errors.Annotatef(err, "grant lease")
	}

	keepAlive, err := c.rawClient.KeepAlive(c.ctx, lease.ID)
	if err != nil || keepAlive == nil {
		c.rawClient.Revoke(c.ctx, lease.ID)
		return errors.Annotate(err, "keep alive lease")
	}

	err = c.put(k, v, clientv3.WithLease(lease.ID))
	if err != nil {
		return errors.Annotate(err, "put k/v with lease")
	}
	return nil
}

// because this method will be called by more than one goroutine
// this method will hold clientset lock
func (c *clientSet) clean() {
	c.lock.Lock()
	if c.gxClient != nil {

		// close gx client, it will close raw etcdv3 client
		c.gxClient.Close()
		c.gxClient = nil
		c.rawClient = nil

		// cancel all context
		c.cancel()
		c.ctx = nil

		c.lock.Unlock()
		return
	}
	c.lock.Unlock()
}

type Client struct {
	name      string
	endpoints []string
	timeout   time.Duration

	sync.Mutex // for control clientSet && event registry
	cs         *clientSet

	done          chan struct{}
	Wait          sync.WaitGroup
	eventRegistry map[string][]clientv3.WatchChan
}

type Options struct {
	name      string
	endpoints []string
	client    *Client
}

type Option func(*Options)

func WithEndpoints(endpoints ...string) Option {
	return func(opt *Options) {
		opt.endpoints = endpoints
	}
}
func WithName(name string) Option {
	return func(opt *Options) {
		opt.name = name
	}
}

func StateToString(state connectivity.State) string {
	switch state {
	case connectivity.Shutdown:
		return "etcdv3 disconnected"
	case connectivity.TransientFailure:
		return "etcdv3 transient failure"
	case connectivity.Idle:
		return "etcdv3 connect idle"
	case connectivity.Ready:
		return "etcdv3 client ready"
	case connectivity.Connecting:
		return "etcdv3 client connecting"
	default:
		return state.String()
	}

	return "etcdv3 unknown state"
}

func ValidateClient(container clientFacade, opts ...Option) error {
	var (
		err error
	)
	options := &Options{}
	for _, opt := range opts {
		opt(options)
	}

	err = nil

	lock := container.ClientLock()
	url := container.GetUrl()

	lock.Lock()
	defer lock.Unlock()

	// bootstrap all clientset
	if container.Client() == nil {
		//in dubbp ,every registry only connect one node ,so this is []string{r.Address}
		timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
		if err != nil {
			logger.Errorf("timeout config %v is invalid ,err is %v",
				url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT), err.Error())
			return errors.Annotatef(err, "newETCDV3Client(address:%+v)", url.Location)
		}
		newClient, err := newClient(options.name, []string{url.Location}, timeout)
		if err != nil {
			logger.Warnf("newETCDV3Client(name{%s}, etcd addresss{%v}, timeout{%d}) = error{%v}",
				options.name, url.Location, timeout.String(), err)
			return errors.Annotatef(err, "newETCDV3Client(address:%+v)", url.Location)
		}
		container.SetClient(newClient)
	}

	if container.Client().cs == nil {

		err = newClientSet(container.Client().endpoints, container.Client().timeout, container.Client())
		if err != nil {
			return errors.Annotate(err, "new client set")
		}
		container.Client().cs.startMaintenanceChan <- struct{}{}
	}

	return errors.Annotatef(err, "newETCDV3Client(address:%+v)", url.PrimitiveURL)
}

func newClient(name string, endpoints []string, timeout time.Duration) (*Client, error) {

	var (
		err error
		out *Client
	)
	out = &Client{
		name:          name,
		endpoints:     endpoints,
		timeout:       timeout,
		done:          make(chan struct{}),
		eventRegistry: make(map[string][]clientv3.WatchChan),
	}

	err = newClientSet(endpoints, timeout, out)
	if err != nil {
		return nil, errors.Annotate(err, "new client set")
	}

	// start maintenanceChan
	out.cs.startMaintenanceChan <- struct{}{}
	return out, nil
}

func (c *Client) stop() bool {
	select {
	case <-c.done:
		return true
	default:
		close(c.done)
	}

	return false
}
func (c *Client) RegisterEvent(key string, wc chan clientv3.WatchResponse) {

	if key == "" || wc == nil {
		return
	}

	c.Lock()
	a := c.eventRegistry[key]
	a = append(a, wc)
	c.eventRegistry[key] = a

	go func() {
		wcc := c.cs.rawClient.Watch(c.cs.ctx, key)
		for msg := range wcc {
			wc <- msg
		}
		// when wcc close, close the wc
		close(wc)
	}()

	logger.Debugf("etcdClient{%s} register event{path:%s, ptr:%p}", c.name, key, wc)
	c.Unlock()
}

func (c *Client) UnregisterEvent(key string, event chan clientv3.WatchResponse) {

	if key == "" {
		return
	}

	c.Lock()
	defer c.Unlock()
	infoList, ok := c.eventRegistry[key]
	if !ok {
		return
	}
	for i, e := range infoList {
		if e == event {
			arr := infoList
			infoList = append(arr[:i], arr[i+1:]...)
			logger.Debugf("etcdClient{%s} unregister event{path:%s, event:%p}", c.name, key, event)
		}
	}
	logger.Debugf("after etcdClient{%s} unregister event{path:%s, event:%p}, array length %d",
		c.name, key, event, len(infoList))
	if len(infoList) == 0 {
		delete(c.eventRegistry, key)
	} else {
		c.eventRegistry[key] = infoList
	}
}

func (c *Client) Done() <-chan struct{} {
	return c.done
}

func (c *Client) Valid() bool {
	select {
	case <-c.done:
		return false
	default:
	}

	valid := true
	c.Lock()
	if c.cs == nil {
		valid = false
	}
	c.Unlock()

	return valid
}

func (c *Client) Close() {
	if c == nil {
		return
	}

	c.stop()
	c.Wait.Wait()
	c.Lock()
	if c.cs != nil {
		c.cs.clean()
		c.cs = nil
	}
	c.Unlock()
	logger.Warnf("etcd client{name:%s, etcd addr:%s} exit now.", c.name, c.endpoints)
}

func (c *Client) Create(k string, v string) error {

	err := ErrNilETCDV3ClientConn

	c.Lock()
	if c.cs != nil {
		err = c.cs.put(k, v)
	}
	c.Unlock()
	return errors.Annotatef(err, "etcd client put key %s value %s", k, v)
}

func (c *Client) Delete(key string) error {

	err := ErrNilETCDV3ClientConn
	c.Lock()
	if c.cs != nil {
		err = c.cs.delete(key)
	}
	c.Unlock()
	return errors.Annotatef(err, "etcd client delete (basePath:%s)", key)
}

func (c *Client) RegisterTemp(basePath string, node string) (string, error) {

	err := ErrNilETCDV3ClientConn
	completePath := path.Join(basePath, node)
	c.Lock()
	if c.cs != nil {
		err = c.cs.keepAliveKV(completePath, "")
	}
	c.Unlock()
	logger.Debugf("etcdClient{%s} create a tmp node:%s\n", c.name, completePath)

	if err != nil {
		return "", errors.Annotatef(err, "etcd client create tmp k %s", completePath)
	}

	return completePath, nil
}

func (c *Client) GetChildrenW(path string) ([]string, clientv3.WatchChan, error) {

	var (
		children []string
		err      error
		wc       clientv3.WatchChan
	)
	err = ErrNilETCDV3ClientConn
	c.Lock()
	if c.cs != nil {
		children, _, wc, err = c.cs.getChildrenW(path)
	}
	c.Unlock()
	if err != nil {
		if errors.Cause(err) == ErrKVPairNotFound {
			return nil, nil, errors.Annotatef(err,"path{%s} has none children", path)
		}
		logger.Errorf("etcdv3.ChildrenW(path{%s}) = error(%v)", path, err)
		return nil, nil, errors.Annotatef(err, "etcdv3.ChildrenW(path:%s)", path)
	}

	return children, wc, nil
}

func (c *Client) GetChildren(path string) ([]string, error) {
	var (
		err      error
		children []string
	)

	err = ErrNilETCDV3ClientConn
	c.Lock()
	if c.cs != nil {
		children, _, err = c.cs.getChildren(path)
	}
	c.Unlock()
	if err != nil {
		if errors.Cause(err) == ErrKVPairNotFound {
			return nil, errors.Annotatef(err,"path{%s} has none children", path)
		}
		logger.Errorf("clientv3.Children(path{%s}) = error(%v)", path, perrors.WithStack(err))
		return nil, errors.Annotatef(err, "clientv3.Children(path:%s)", path)
	}
	return children, nil
}

func (c *Client) ExistW(path string) (clientv3.WatchChan, error) {

	var (
		err = ErrNilETCDV3ClientConn
		out clientv3.WatchChan
	)

	c.Lock()
	if c.cs != nil {
		_, err = c.cs.watch(path)
	}
	c.Unlock()
	if err != nil {
		if errors.Cause(err) == ErrKVPairNotFound {
			return nil, errors.Annotatef(err, "path{%s} not exist", path)
		}
		return nil, errors.Annotatef(err, "clientv3.ExistW(path:%s)", path)
	}

	return out, nil
}

func (c *Client) GetContent(path string) ([]byte, error) {

	c.Lock()
	value, err := c.cs.get(path)
	if err != nil {
		return nil, errors.Annotatef(err, "client set get: %s", path)
	}
	c.Unlock()

	return []byte(value), nil
}
