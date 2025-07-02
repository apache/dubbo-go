package etcd

//
///*
//   cache_listener.go – etcd-specific implementation of CacheListener
//   This version no longer depends on Zookeeper types.  It provides:
//   • AddListener / RemoveListener  – register & remove per-key listeners
//   • PushEvent                    – called by etcdDynamicConfiguration when a Watch event arrives
//   • Local snapshot               – sync.Map[key] -> map[listener]struct{}
//*/
//
//import (
//	"strings"
//	"sync"
//
//	"dubbo.apache.org/dubbo-go/v3/common/constant"
//	"dubbo.apache.org/dubbo-go/v3/config_center"
//	"dubbo.apache.org/dubbo-go/v3/metrics"
//	metricsConfigCenter "dubbo.apache.org/dubbo-go/v3/metrics/config_center"
//	"dubbo.apache.org/dubbo-go/v3/remoting"
//)
//
//// CacheListener keeps a local snapshot of the latest config for each key and
//// fans out change events to multiple ConfigurationListeners.
//type CacheListener struct {
//	keyListeners sync.Map // key -> map[ConfigurationListener]struct{}
//	rootPath     string
//}
//
//func NewCacheListener(rootPath string) *CacheListener {
//	return &CacheListener{rootPath: rootPath}
//}
//
//// AddListener registers a ConfigurationListener for a qualified etcd key.
//func (cl *CacheListener) AddListener(key string, listener config_center.ConfigurationListener) {
//	lst, _ := cl.keyListeners.LoadOrStore(key, map[config_center.ConfigurationListener]struct{}{listener: {}})
//	lst.(map[config_center.ConfigurationListener]struct{})[listener] = struct{}{}
//	cl.keyListeners.Store(key, lst)
//}
//
//// RemoveListener removes a previously registered listener.
//func (cl *CacheListener) RemoveListener(key string, listener config_center.ConfigurationListener) {
//	if v, ok := cl.keyListeners.Load(key); ok {
//		delete(v.(map[config_center.ConfigurationListener]struct{}), listener)
//	}
//}
//
//// PushEvent is invoked by etcdDynamicConfiguration when the watched key changes.
//func (cl *CacheListener) PushEvent(qualifiedKey, value string) {
//	key, group := cl.pathToKeyGroup(qualifiedKey)
//
//	// publish metric
//	metrics.Publish(metricsConfigCenter.NewIncMetricEvent(key, group, string(remoting.EventTypeUpdate), metricsConfigCenter.Etcd))
//
//	if v, ok := cl.keyListeners.Load(qualifiedKey); ok {
//		for l := range v.(map[config_center.ConfigurationListener]struct{}) {
//			l.Process(&config_center.ConfigChangeEvent{
//				Key:        key,
//				Value:      value,
//				ConfigType: remoting.EventTypeUpdate,
//			})
//		}
//	}
//}
//
//// pathToKeyGroup converts an etcd full key into {key, group} pair following
//// the /dubbo/config/<namespace>/<group>/key convention.
//func (cl *CacheListener) pathToKeyGroup(path string) (string, string) {
//	if path == "" {
//		return path, ""
//	}
//	groupKey := strings.TrimPrefix(path, cl.rootPath+constant.PathSeparator)
//	groupKey = strings.ReplaceAll(groupKey, constant.PathSeparator, constant.DotSeparator)
//	idx := strings.Index(groupKey, constant.DotSeparator)
//	if idx == -1 {
//		return groupKey, ""
//	}
//	return groupKey[idx+1:], groupKey[:idx]
//}
