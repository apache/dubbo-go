package etcd

import (
	"context"

	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting"
	"github.com/juju/errors"
)

type dataListener struct {
	interestedURL []*common.URL
	listener      *configurationListener
}

func NewRegistryDataListener(listener *configurationListener) *dataListener {
	return &dataListener{listener: listener, interestedURL: []*common.URL{}}
}

func (l *dataListener) AddInterestedURL(url *common.URL) {
	l.interestedURL = append(l.interestedURL, url)
}

func (l *dataListener) DataChange(eventType remoting.Event) bool {
	serviceURL, err := common.NewURL(context.TODO(), eventType.Content)
	if err != nil {
		logger.Errorf("Listen NewURL(r{%s}) = error{%v}", eventType.Content, err)
		return false
	}
	for _, v := range l.interestedURL {
		if serviceURL.URLEqual(*v) {
			l.listener.Process(&remoting.ConfigChangeEvent{Value: serviceURL, ConfigType: eventType.Action})
			return true
		}
	}

	return false
}

type configurationListener struct {
	registry *etcdV3Registry
	events   chan *remoting.ConfigChangeEvent
}

func NewConfigurationListener(reg *etcdV3Registry) *configurationListener {
	return &configurationListener{registry: reg, events: make(chan *remoting.ConfigChangeEvent, 32)}
}
func (l *configurationListener) Process(configType *remoting.ConfigChangeEvent) {
	l.events <- configType
}

func (l *configurationListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-l.registry.done:
			logger.Warnf("listener's etcd client connection is broken, so etcd event listener exit now.")
			return nil, errors.New("listener stopped")

		case e := <-l.events:
			logger.Debugf("got etcd event %s", e)
			if e.ConfigType == remoting.EventTypeDel {
				select {
				case <-l.registry.done:
					logger.Warnf("update @result{%s}. But its connection to registry is invalid", e.Value)
				default:
				}
				continue
			}
			//r.update(e.res)
			//write to invoker
			//r.outerEventCh <- e.res
			return &registry.ServiceEvent{Action: e.ConfigType, Service: e.Value.(common.URL)}, nil
		}
	}
}
func (l *configurationListener) Close() {
	l.registry.wg.Done()
}
