package nacos

import (
	"bytes"
	"net/url"
	"reflect"
	"strconv"
	"sync"
)

import (
	"github.com/nacos-group/nacos-sdk-go/clients/naming_client"
	"github.com/nacos-group/nacos-sdk-go/model"
	"github.com/nacos-group/nacos-sdk-go/vo"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting"
)

type nacosListener struct {
	sync.Mutex
	namingClient    naming_client.INamingClient
	listenUrl       common.URL
	events          chan *remoting.ConfigChangeEvent
	hostMapInstance map[string]model.Instance
	done            chan struct{}
	subscribeParam  *vo.SubscribeParam
}

func NewNacosListener(url common.URL, namingClient naming_client.INamingClient) (*nacosListener, error) {
	listener := &nacosListener{namingClient: namingClient, listenUrl: url, events: make(chan *remoting.ConfigChangeEvent, 32), hostMapInstance: map[string]model.Instance{}, done: make(chan struct{})}
	err := listener.startListen()
	return listener, err
}

func generateInstance(ss model.SubscribeService) model.Instance {
	return model.Instance{
		InstanceId:  ss.InstanceId,
		Ip:          ss.Ip,
		Port:        ss.Port,
		ServiceName: ss.ServiceName,
		Valid:       ss.Valid,
		Enable:      ss.Enable,
		Weight:      ss.Weight,
		Metadata:    ss.Metadata,
		ClusterName: ss.ClusterName,
	}
}

func generateUrl(instance model.Instance) *common.URL {
	if instance.Metadata == nil {
		logger.Errorf("nacos instance metadata is empty,instance:%+v", instance)
		return nil
	}
	path := instance.Metadata["path"]
	myInterface := instance.Metadata["interface"]
	if path == "" && myInterface == "" {
		logger.Errorf("nacos instance metadata does not have  both path key and interface key,instance:%+v", instance)
		return nil
	}
	if path == "" && myInterface != "" {
		path = "/" + myInterface
	}
	protocol := instance.Metadata["protocol"]
	if protocol == "" {
		logger.Errorf("nacos instance metadata does not have protocol key,instance:%+v", instance)
		return nil
	}
	urlMap := url.Values{}
	for k, v := range instance.Metadata {
		urlMap.Set(k, v)
	}
	return common.NewURLWithOptions(common.WithIp(instance.Ip), common.WithPort(strconv.Itoa(int(instance.Port))), common.WithProtocol(protocol), common.WithParams(urlMap), common.WithPath(path))
}

func (nl *nacosListener) Callback(services []model.SubscribeService, err error) {
	if err != nil {
		logger.Errorf("nacos subscribe callback error:%s ", err.Error())
		return
	}
	nl.Lock()
	defer nl.Unlock()
	var addInstances []model.Instance
	var delInstances []model.Instance
	var updateInstances []model.Instance

	newInstanceMap := map[string]model.Instance{}

	for _, s := range services {
		if !s.Enable || !s.Valid {
			//实例不可以用
			continue
		}
		host := s.Ip + ":" + strconv.Itoa(int(s.Port))
		instance := generateInstance(s)
		newInstanceMap[host] = instance
		if old, ok := nl.hostMapInstance[host]; !ok {
			//新增实例节点
			addInstances = append(addInstances, instance)
		} else {
			//实例更新
			if !reflect.DeepEqual(old, instance) {
				updateInstances = append(updateInstances, instance)
			}
		}
	}

	//判断旧的实例是否在新实例列表中，不存在则代表实例已下线
	for host, inst := range nl.hostMapInstance {
		if _, ok := newInstanceMap[host]; !ok {
			delInstances = append(delInstances, inst)
		}
	}

	nl.hostMapInstance = newInstanceMap

	for _, add := range addInstances {
		newUrl := generateUrl(add)
		if newUrl != nil {
			nl.process(&remoting.ConfigChangeEvent{Value: *newUrl, ConfigType: remoting.EventTypeAdd})
		}
	}
	for _, del := range delInstances {
		newUrl := generateUrl(del)
		if newUrl != nil {
			nl.process(&remoting.ConfigChangeEvent{Value: *newUrl, ConfigType: remoting.EventTypeDel})
		}
	}

	for _, update := range updateInstances {
		newUrl := generateUrl(update)
		if newUrl != nil {
			nl.process(&remoting.ConfigChangeEvent{Value: *newUrl, ConfigType: remoting.EvnetTypeUpdate})
		}
	}
}

func getSubscribeName(url common.URL) string {
	var buffer bytes.Buffer

	buffer.Write([]byte(common.DubboNodes[common.PROVIDER]))
	appendParam(&buffer, url, constant.INTERFACE_KEY)
	appendParam(&buffer, url, constant.VERSION_KEY)
	appendParam(&buffer, url, constant.GROUP_KEY)
	return buffer.String()
}

func (nl *nacosListener) startListen() error {
	if nl.namingClient == nil {
		return perrors.New("nacos naming client stopped")
	}
	serviceName := getSubscribeName(nl.listenUrl)
	nl.subscribeParam = &vo.SubscribeParam{ServiceName: serviceName, SubscribeCallback: nl.Callback}
	return nl.namingClient.Subscribe(nl.subscribeParam)
}

func (nl *nacosListener) stopListen() error {
	return nl.namingClient.Unsubscribe(nl.subscribeParam)
}

func (nl *nacosListener) process(configType *remoting.ConfigChangeEvent) {
	nl.events <- configType
}

func (nl *nacosListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-nl.done:
			logger.Warnf("nacos listener is close!")
			return nil, perrors.New("listener stopped")

		case e := <-nl.events:
			logger.Debugf("got nacos event %s", e)
			return &registry.ServiceEvent{Action: e.ConfigType, Service: e.Value.(common.URL)}, nil
		}
	}
}

func (nl *nacosListener) Close() {
	nl.stopListen()
	close(nl.done)
}
