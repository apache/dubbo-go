package etcdv3

import (
	"context"
	"path"
	"sync"
	"time"
)

import (
	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/clientv3/concurrency"
	perrors "github.com/pkg/errors"
	"google.golang.org/grpc"
)

import (
	"github.com/apache/dubbo-go/common/logger"
)

const (
	ConnDelay            = 3
	MaxFailTimes         = 15
	RegistryETCDV3Client = "etcd registry"
)

var (
	ErrNilETCDV3Client = perrors.New("etcd raw client is nil") // full describe the ERR
	ErrKVPairNotFound  = perrors.New("k/v pair not found")
)

type Options struct {
	name      string
	endpoints []string
	client    *Client
	timeout   time.Duration
	heartbeat int // heartbeat second
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
func WithTimeout(timeout time.Duration) Option {
	return func(opt *Options) {
		opt.timeout = timeout
	}
}

func WithHeartbeat(heartbeat int) Option {
	return func(opt *Options) {
		opt.heartbeat = heartbeat
	}
}

func ValidateClient(container clientFacade, opts ...Option) error {

	options := &Options{
		heartbeat: 1, // default heartbeat
	}
	for _, opt := range opts {
		opt(options)
	}

	lock := container.ClientLock()
	lock.Lock()
	defer lock.Unlock()

	// new Client
	if container.Client() == nil {
		newClient, err := newClient(options.name, options.endpoints, options.timeout, options.heartbeat)
		if err != nil {
			logger.Warnf("new etcd client (name{%s}, etcd addresses{%v}, timeout{%d}) = error{%v}",
				options.name, options.endpoints, options.timeout, err)
			return perrors.WithMessagef(err, "new client (address:%+v)", options.endpoints)
		}
		container.SetClient(newClient)
	}

	// Client lose connection with etcd server
	if container.Client().rawClient == nil {

		newClient, err := newClient(options.name, options.endpoints, options.timeout, options.heartbeat)
		if err != nil {
			logger.Warnf("new etcd client (name{%s}, etcd addresses{%v}, timeout{%d}) = error{%v}",
				options.name, options.endpoints, options.timeout, err)
			return perrors.WithMessagef(err, "new client (address:%+v)", options.endpoints)
		}
		container.SetClient(newClient)
	}

	return nil
}

type Client struct {
	lock sync.RWMutex

	// these properties are only set once when they are started.
	name      string
	endpoints []string
	timeout   time.Duration
	heartbeat int

	ctx       context.Context    // if etcd server connection lose, the ctx.Done will be sent msg
	cancel    context.CancelFunc // cancel the ctx,  all watcher will stopped
	rawClient *clientv3.Client

	exit chan struct{}
	Wait sync.WaitGroup
}

func newClient(name string, endpoints []string, timeout time.Duration, heartbeat int) (*Client, error) {

	ctx, cancel := context.WithCancel(context.Background())
	rawClient, err := clientv3.New(clientv3.Config{
		Context:     ctx,
		Endpoints:   endpoints,
		DialTimeout: timeout,
		DialOptions: []grpc.DialOption{grpc.WithBlock()},
	})
	if err != nil {
		return nil, perrors.WithMessage(err, "new raw client block connect to server")
	}

	c := &Client{

		name:      name,
		timeout:   timeout,
		endpoints: endpoints,
		heartbeat: heartbeat,

		ctx:       ctx,
		cancel:    cancel,
		rawClient: rawClient,

		exit: make(chan struct{}),
	}

	if err := c.maintenanceStatus(); err != nil {
		return nil, perrors.WithMessage(err, "client maintenance status")
	}
	return c, nil
}

// NOTICE: need to get the lock before calling this method
func (c *Client) clean() {

	// close raw client
	c.rawClient.Close()

	// cancel ctx for raw client
	c.cancel()

	// clean raw client
	c.rawClient = nil
}

func (c *Client) stop() bool {

	select {
	case <-c.exit:
		return true
	default:
		close(c.exit)
	}
	return false
}

func (c *Client) Close() {

	if c == nil {
		return
	}

	// stop the client
	c.stop()

	// wait client maintenance status stop
	c.Wait.Wait()

	c.lock.Lock()
	if c.rawClient != nil {
		c.clean()
	}
	c.lock.Unlock()
	logger.Warnf("etcd client{name:%s, endpoints:%s} exit now.", c.name, c.endpoints)
}

func (c *Client) maintenanceStatus() error {

	s, err := concurrency.NewSession(c.rawClient, concurrency.WithTTL(c.heartbeat))
	if err != nil {
		return perrors.WithMessage(err, "new session with server")
	}

	// must add wg before go maintenance status goroutine
	c.Wait.Add(1)
	go c.maintenanceStatusLoop(s)
	return nil
}

func (c *Client) maintenanceStatusLoop(s *concurrency.Session) {

	defer func() {
		c.Wait.Done()
		logger.Infof("etcd client {endpoints:%v, name:%s} maintenance goroutine game over.", c.endpoints, c.name)
	}()

	for {
		select {
		case <-c.Done():
			// Client be stopped, will clean the client hold resources
			return
		case <-s.Done():
			logger.Warn("etcd server stopped")
			c.lock.Lock()
			// when etcd server stopped, cancel ctx, stop all watchers
			c.clean()
			// when connection lose, stop client, trigger reconnect to etcd
			c.stop()
			c.lock.Unlock()
			return
		}
	}
}

// if k not exist will put k/v in etcd
// if k is already exist in etcd, return nil
func (c *Client) put(k string, v string, opts ...clientv3.OpOption) error {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return ErrNilETCDV3Client
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

func (c *Client) delete(k string) error {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return ErrNilETCDV3Client
	}

	_, err := c.rawClient.Delete(c.ctx, k)
	if err != nil {
		return err

	}
	return nil
}

func (c *Client) get(k string) (string, error) {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return "", ErrNilETCDV3Client
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

func (c *Client) CleanKV() error {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return ErrNilETCDV3Client
	}

	_, err := c.rawClient.Delete(c.ctx, "", clientv3.WithPrefix())
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) getChildren(k string) ([]string, []string, error) {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return nil, nil, ErrNilETCDV3Client
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

func (c *Client) watchWithPrefix(prefix string) (clientv3.WatchChan, error) {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return nil, ErrNilETCDV3Client
	}

	return c.rawClient.Watch(c.ctx, prefix, clientv3.WithPrefix()), nil
}

func (c *Client) watch(k string) (clientv3.WatchChan, error) {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return nil, ErrNilETCDV3Client
	}

	return c.rawClient.Watch(c.ctx, k), nil
}

func (c *Client) keepAliveKV(k string, v string) error {

	c.lock.RLock()
	defer c.lock.RUnlock()

	if c.rawClient == nil {
		return ErrNilETCDV3Client
	}

	lease, err := c.rawClient.Grant(c.ctx, int64(time.Second.Seconds()))
	if err != nil {
		return perrors.WithMessage(err, "grant lease")
	}

	keepAlive, err := c.rawClient.KeepAlive(c.ctx, lease.ID)
	if err != nil || keepAlive == nil {
		c.rawClient.Revoke(c.ctx, lease.ID)
		return perrors.WithMessage(err, "keep alive lease")
	}

	_, err = c.rawClient.Put(c.ctx, k, v, clientv3.WithLease(lease.ID))
	if err != nil {
		return perrors.WithMessage(err, "put k/v with lease")
	}
	return nil
}

func (c *Client) Done() <-chan struct{} {
	return c.exit
}

func (c *Client) Valid() bool {
	select {
	case <-c.exit:
		return false
	default:
	}

	c.lock.RLock()
	if c.rawClient == nil {
		c.lock.RUnlock()
		return false
	}
	c.lock.RUnlock()
	return true
}

func (c *Client) Create(k string, v string) error {

	err := c.put(k, v)
	if err != nil {
		return perrors.WithMessagef(err, "put k/v (key: %s value %s)", k, v)
	}
	return nil
}

func (c *Client) Delete(k string) error {

	err := c.delete(k)
	if err != nil {
		return perrors.WithMessagef(err, "delete k/v (key %s)", k)
	}

	return nil
}

func (c *Client) RegisterTemp(basePath string, node string) (string, error) {

	completeKey := path.Join(basePath, node)

	err := c.keepAliveKV(completeKey, "")
	if err != nil {
		return "", perrors.WithMessagef(err, "keepalive kv (key %s)", completeKey)
	}

	return completeKey, nil
}

func (c *Client) GetChildrenKVList(k string) ([]string, []string, error) {

	kList, vList, err := c.getChildren(k)
	if err != nil {
		return nil, nil, perrors.WithMessagef(err, "get key children (key %s)", k)
	}
	return kList, vList, nil
}

func (c *Client) Get(k string) (string, error) {

	v, err := c.get(k)
	if err != nil {
		return "", perrors.WithMessagef(err, "get key value (key %s)", k)
	}

	return v, nil
}

func (c *Client) Watch(k string) (clientv3.WatchChan, error) {

	wc, err := c.watch(k)
	if err != nil {
		return nil, perrors.WithMessagef(err, "watch prefix (key %s)", k)
	}
	return wc, nil
}

func (c *Client) WatchWithPrefix(prefix string) (clientv3.WatchChan, error) {

	wc, err := c.watchWithPrefix(prefix)
	if err != nil {
		return nil, perrors.WithMessagef(err, "watch prefix (key %s)", prefix)
	}
	return wc, nil
}
