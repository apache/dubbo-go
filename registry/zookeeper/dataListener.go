package zookeeper

import (
	"context"
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
	zk "github.com/apache/dubbo-go/remoting/zookeeper"
	perrors "github.com/pkg/errors"
)

type RegistryDataListener struct {
	interestedURL []*common.URL
	listener      *RegistryConfigurationListener
}

func NewRegistryDataListener(listener *RegistryConfigurationListener) *RegistryDataListener {
	return &RegistryDataListener{listener: listener, interestedURL: []*common.URL{}}
}
func (l *RegistryDataListener) AddInterestedURL(url *common.URL) {
	l.interestedURL = append(l.interestedURL, url)
}

func (l *RegistryDataListener) DataChange(eventType zk.ZkEvent) bool {
	serviceURL, err := common.NewURL(context.TODO(), eventType.Res.Content)
	if err != nil {
		logger.Errorf("Listen NewURL(r{%s}) = error{%v}", eventType.Res.Content, err)
		return false
	}
	for _, v := range l.interestedURL {
		if serviceURL.URLEqual(*v) {
			l.listener.Process(&common.ConfigChangeEvent{Value: serviceURL, ConfigType: eventType.Res.Action})
			return true
		}
	}

	return false
}

type RegistryConfigurationListener struct {
	client   *zk.ZookeeperClient
	registry *zkRegistry
	events   chan *common.ConfigChangeEvent
}

func NewRegistryConfigurationListener(client *zk.ZookeeperClient, reg *zkRegistry) *RegistryConfigurationListener {
	reg.wg.Add(1)
	return &RegistryConfigurationListener{client: client, registry: reg, events: make(chan *common.ConfigChangeEvent, 32)}
}
func (l *RegistryConfigurationListener) Process(configType *common.ConfigChangeEvent) {
	l.events <- configType
}

func (l *RegistryConfigurationListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-l.client.Done():
			logger.Warnf("listener's zk client connection is broken, so zk event listener exit now.")
			return nil, perrors.New("listener stopped")

		case <-l.registry.done:
			logger.Warnf("zk consumer register has quit, so zk event listener exit asap now.")
			return nil, perrors.New("listener stopped")

		case e := <-l.events:
			logger.Debugf("got zk event %s", e)
			if e.ConfigType == common.Del && !l.valid() {
				logger.Warnf("update @result{%s}. But its connection to registry is invalid", e.Value)
				continue
			}
			//r.update(e.res)
			//write to invoker
			//r.outerEventCh <- e.res
			return &registry.ServiceEvent{Action: e.ConfigType, Service: e.Value.(common.URL)}, nil
		}
	}
}
func (l *RegistryConfigurationListener) Close() {
	l.registry.wg.Done()
}

func (l *RegistryConfigurationListener) valid() bool {
	return l.client.ZkConnValid()
}
