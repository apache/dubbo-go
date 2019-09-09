package etcdv3

import (
	"context"
	"strings"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting"
)

type dataListener struct {
	interestedURL []*common.URL
	listener      config_center.ConfigurationListener
}

func NewRegistryDataListener(listener config_center.ConfigurationListener) *dataListener {
	return &dataListener{listener: listener, interestedURL: []*common.URL{}}
}

func (l *dataListener) AddInterestedURL(url *common.URL) {
	l.interestedURL = append(l.interestedURL, url)
}

func (l *dataListener) DataChange(eventType remoting.Event) bool {

	url := eventType.Path[strings.Index(eventType.Path, "/providers/")+len("/providers/"):]
	serviceURL, err := common.NewURL(context.Background(), url)
	if err != nil {
		logger.Warnf("Listen NewURL(r{%s}) = error{%v}", eventType.Path, err)
		return false
	}

	for _, v := range l.interestedURL {
		if serviceURL.URLEqual(*v) {
			l.listener.Process(
				&config_center.ConfigChangeEvent{
					Key:        eventType.Path,
					Value:      serviceURL,
					ConfigType: eventType.Action,
				},
			)
			return true
		}
	}

	return false
}

type configurationListener struct {
	registry *etcdV3Registry
	events   chan *config_center.ConfigChangeEvent
}

func NewConfigurationListener(reg *etcdV3Registry) *configurationListener {
	// add a new waiter
	reg.wg.Add(1)
	return &configurationListener{registry: reg, events: make(chan *config_center.ConfigChangeEvent, 32)}
}
func (l *configurationListener) Process(configType *config_center.ConfigChangeEvent) {
	l.events <- configType
}

func (l *configurationListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-l.registry.done:
			logger.Warnf("listener's etcd client connection is broken, so etcd event listener exit now.")
			return nil, perrors.New("listener stopped")

		case e := <-l.events:
			logger.Infof("got etcd event %#v", e)
			if e.ConfigType == remoting.EventTypeDel {
				select {
				case <-l.registry.done:
					logger.Warnf("update @result{%s}. But its connection to registry is invalid", e.Value)
				default:
				}
				continue
			}
			return &registry.ServiceEvent{Action: e.ConfigType, Service: e.Value.(common.URL)}, nil
		}
	}
}
func (l *configurationListener) Close() {
	l.registry.wg.Done()
}
