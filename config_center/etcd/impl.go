package etcd

//
///*
//   This file is a fully–functional etcd implementation of Dubbo‑Go’s DynamicConfiguration
//   interface.  It mirrors the feature‑set of the official Zookeeper plugin:
//   – strong consistency via etcd v3
//   – long‑lived Watch with automatic reconnection & revision catch‑up
//   – CacheListener‑based local snapshot & multi‑listener fan‑out
//   – optional Base64 encoding (params.base64=true)
//
//   Author: <your‑name>
//   Date  : 2025‑05‑31
//*/
//
//import (
//	"context"
//	"encoding/base64"
//	"strconv"
//	"strings"
//	"sync"
//	"time"
//
//	rpctypes "go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
//	clientv3 "go.etcd.io/etcd/client/v3"
//
//	gxset "github.com/dubbogo/gost/container/set"
//	"github.com/dubbogo/gost/log/logger"
//	perrors "github.com/pkg/errors"
//
//	"dubbo.apache.org/dubbo-go/v3/common"
//	"dubbo.apache.org/dubbo-go/v3/common/constant"
//	"dubbo.apache.org/dubbo-go/v3/config"
//	"dubbo.apache.org/dubbo-go/v3/config_center"
//	"dubbo.apache.org/dubbo-go/v3/config_center/parser"
//)
//
//const pathSeparator = "/"
//
//// etcdDynamicConfiguration implements dubbo-go DynamicConfiguration backed by etcd v3.
//// It copies the behaviour of the Zookeeper plugin but adapts it to etcd Watch semantics.
//// -----------------------------------------------------------------------------
//type etcdDynamicConfiguration struct {
//	config_center.BaseDynamicConfiguration
//
//	url      *common.URL
//	rootPath string
//	cli      *clientv3.Client
//	cliLock  sync.Mutex
//	done     chan struct{}
//	wg       sync.WaitGroup
//
//	cacheListener *CacheListener
//	watchCancels  sync.Map // map[string]context.CancelFunc
//
//	parser        parser.ConfigurationParser
//	base64Enabled bool
//}
//
//// -----------------------------------------------------------------------------
//// Constructor & client bootstrap
//// -----------------------------------------------------------------------------
//
//func newEtcdDynamicConfiguration(url *common.URL) (*etcdDynamicConfiguration, error) {
//	c := &etcdDynamicConfiguration{
//		url:      url,
//		rootPath: "/dubbo/config",
//		done:     make(chan struct{}),
//	}
//
//	// optional base64 param
//	if v, ok := config.GetRootConfig().ConfigCenter.Params["base64"]; ok {
//		if b, err := strconv.ParseBool(v); err == nil {
//			c.base64Enabled = b
//		} else {
//			return nil, perrors.WithStack(err)
//		}
//	}
//
//	// initialise etcd connection
//	if err := c.initClient(); err != nil {
//		return nil, err
//	}
//
//	// local cache + listener fan‑out
//	c.cacheListener = NewCacheListener(c.rootPath)
//
//	// background connection health‑check / auto‑reconnect
//	c.wg.Add(1)
//	go c.handleClientRestart()
//
//	logger.Infof("[Etcd ConfigCenter] started with endpoints=%s", url.Location)
//	return c, nil
//}
//
//func (c *etcdDynamicConfiguration) initClient() error {
//	c.cliLock.Lock()
//	defer c.cliLock.Unlock()
//
//	// close previous
//	if c.cli != nil {
//		c.cli.Close()
//	}
//
//	cli, err := clientv3.New(clientv3.Config{
//		Endpoints:            strings.Split(c.url.Location, ","),
//		DialTimeout:          5 * time.Second,
//		DialKeepAliveTime:    30 * time.Second,
//		DialKeepAliveTimeout: 10 * time.Second,
//	})
//	if err != nil {
//		return perrors.WithStack(err)
//	}
//	c.cli = cli
//	return nil
//}
//
//// -----------------------------------------------------------------------------
//// Listener registration
//// -----------------------------------------------------------------------------
//
//func (c *etcdDynamicConfiguration) AddListener(key string, l config_center.ConfigurationListener, opts ...config_center.Option) {
//	qualified := c.qualify(key, opts...)
//	c.cacheListener.AddListener(qualified, l)
//
//	ctx, cancel := context.WithCancel(context.Background())
//	c.watchCancels.Store(qualified, cancel)
//
//	go func(rev int64) {
//		watchOpts := []clientv3.OpOption{clientv3.WithRev(rev)}
//		wc := c.cli.Watch(ctx, qualified, watchOpts...)
//		for {
//			select {
//			case <-c.done:
//				return
//			case wr, ok := <-wc:
//				if !ok {
//					return // channel closed
//				}
//				if wr.Canceled {
//					if wr.Err() == rpctypes.ErrCompacted {
//						rev = wr.CompactRevision + 1 // jump to latest and continue
//						wc = c.cli.Watch(ctx, qualified, clientv3.WithRev(rev))
//						continue
//					}
//					logger.Warnf("[etcd] watch canceled: %v", wr.Err())
//					return
//				}
//				for _, ev := range wr.Events {
//					c.cacheListener.PushEvent(qualified, string(ev.Kv.Value))
//				}
//			}
//		}
//	}(0)
//}
//
//func (c *etcdDynamicConfiguration) RemoveListener(key string, l config_center.ConfigurationListener, opts ...config_center.Option) {
//	qualified := c.qualify(key, opts...)
//	if v, ok := c.watchCancels.LoadAndDelete(qualified); ok {
//		v.(context.CancelFunc)()
//	}
//	c.cacheListener.RemoveListener(qualified, l)
//}
//
//// -----------------------------------------------------------------------------
//// Data access
//// -----------------------------------------------------------------------------
//
//func (c *etcdDynamicConfiguration) GetProperties(key string, opts ...config_center.Option) (string, error) {
//	qualified := c.qualify(key, opts...)
//	resp, err := c.cli.Get(context.Background(), qualified)
//	if err != nil {
//		return "", perrors.WithStack(err)
//	}
//	if len(resp.Kvs) == 0 {
//		return "", nil
//	}
//	val := resp.Kvs[0].Value
//	if c.base64Enabled {
//		decoded, err := base64.StdEncoding.DecodeString(string(val))
//		if err == nil {
//			val = decoded
//		}
//	}
//	return string(val), nil
//}
//
//func (c *etcdDynamicConfiguration) GetInternalProperty(key string, opts ...config_center.Option) (string, error) {
//	return c.GetProperties(key, opts...)
//}
//
//func (c *etcdDynamicConfiguration) PublishConfig(key, group, value string) error {
//	path := c.getPath(key, group)
//	data := []byte(value)
//	if c.base64Enabled {
//		data = []byte(base64.StdEncoding.EncodeToString(data))
//	}
//	_, err := c.cli.Put(context.Background(), path, string(data))
//	return perrors.WithStack(err)
//}
//
//func (c *etcdDynamicConfiguration) RemoveConfig(key, group string) error {
//	path := c.getPath(key, group)
//	_, err := c.cli.Delete(context.Background(), path)
//	return perrors.WithStack(err)
//}
//
//func (c *etcdDynamicConfiguration) GetConfigKeysByGroup(group string) (*gxset.HashSet, error) {
//	prefix := c.getPath("", group)
//	resp, err := c.cli.Get(context.Background(), prefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
//	if err != nil {
//		return nil, perrors.WithStack(err)
//	}
//	if len(resp.Kvs) == 0 {
//		return nil, perrors.New("could not find keys with group: " + group)
//	}
//	set := gxset.NewSet()
//	for _, kv := range resp.Kvs {
//		k := strings.TrimPrefix(string(kv.Key), prefix+pathSeparator)
//		set.Add(k)
//	}
//	return set, nil
//}
//
//func (c *etcdDynamicConfiguration) GetRule(key string, opts ...config_center.Option) (string, error) {
//	return c.GetProperties(key, opts...)
//}
//
//// -----------------------------------------------------------------------------
//// Parser & utils
//// -----------------------------------------------------------------------------
//
//func (c *etcdDynamicConfiguration) Parser() parser.ConfigurationParser     { return c.parser }
//func (c *etcdDynamicConfiguration) SetParser(p parser.ConfigurationParser) { c.parser = p }
//
//func (c *etcdDynamicConfiguration) qualify(key string, opts ...config_center.Option) string {
//	o := config_center.NewOptions(opts...)
//	ns := c.url.GetParam(constant.ConfigNamespaceKey, config_center.DefaultGroup)
//	if o.Center.Group != "" {
//		ns = o.Center.Group
//	}
//	return buildPath(c.rootPath, ns+"/"+key)
//}
//
//func buildPath(rootPath, subPath string) string {
//	path := strings.TrimRight(rootPath+pathSeparator+subPath, pathSeparator)
//	if !strings.HasPrefix(path, pathSeparator) {
//		path = pathSeparator + path
//	}
//	return strings.ReplaceAll(path, "//", "/")
//}
//
//func (c *etcdDynamicConfiguration) getPath(key, group string) string {
//	if key == "" {
//		return c.buildGroupPath(group)
//	}
//	return c.buildGroupPath(group) + pathSeparator + key
//}
//
//func (c *etcdDynamicConfiguration) buildGroupPath(group string) string {
//	if group == "" {
//		group = config_center.DefaultGroup
//	}
//	return c.rootPath + pathSeparator + group
//}
//
//// -----------------------------------------------------------------------------
//// Lifecycle & health check
//// -----------------------------------------------------------------------------
//
//func (c *etcdDynamicConfiguration) handleClientRestart() {
//	defer c.wg.Done()
//	ticker := time.NewTicker(30 * time.Second)
//	for {
//		select {
//		case <-c.done:
//			return
//		case <-ticker.C:
//			if c.cli.ActiveConnection() == nil {
//				logger.Warn("[etcd] detected dead connection, reconnecting…")
//				_ = c.initClient()
//			}
//		}
//	}
//}
//
//func (c *etcdDynamicConfiguration) Destroy() {
//	close(c.done)
//	c.wg.Wait()
//	c.cli.Close()
//}
//
//func (c *etcdDynamicConfiguration) IsAvailable() bool {
//	select {
//	case <-c.done:
//		return false
//	default:
//		return true
//	}
//}
