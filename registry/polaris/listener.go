/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package polaris

import (
	"fmt"
	"net/url"
	"strconv"
)

import (
	gxchan "github.com/dubbogo/gost/container/chan"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"

	"github.com/polarismesh/polaris-go/pkg/model"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
)

type polarisNotificationKind uint8

const (
	incrementalNotification polarisNotificationKind = iota
	fullSnapshotNotification
)

type polarisNotification struct {
	kind      polarisNotificationKind
	eventType remoting.EventType
	instances []model.Instance
}

type polarisListener struct {
	watcher       *PolarisServiceWatcher
	events        *gxchan.UnboundedChan
	closeCh       chan struct{}
	pendingEvents []*registry.ServiceEvent
}

// NewPolarisListener new polaris listener
func NewPolarisListener(watcher *PolarisServiceWatcher) (*polarisListener, error) {
	listener := newPolarisListenerState(watcher)
	listener.watcher.AddSubscriber(listener.notify)
	return listener, nil
}

func newPolarisListener(
	watcher *PolarisServiceWatcher,
	initialSnapshot []model.Instance,
	mode initialReconcileMode,
) (*polarisListener, error) {
	listener := newPolarisListenerState(watcher)
	listener.watcher.addRegistrySubscriber(
		initialSnapshot,
		mode,
		listener.notify,
		listener.notifyFullSnapshot,
	)
	return listener, nil
}

func newPolarisListenerState(watcher *PolarisServiceWatcher) *polarisListener {
	return &polarisListener{
		watcher: watcher,
		events:  gxchan.NewUnboundedChan(32),
		closeCh: make(chan struct{}),
	}
}

func (pl *polarisListener) notify(et remoting.EventType, ins []model.Instance) {
	if len(ins) == 0 {
		return
	}
	pl.events.In() <- &polarisNotification{
		kind:      incrementalNotification,
		eventType: et,
		instances: copyInstances(ins),
	}
}

func (pl *polarisListener) notifyFullSnapshot(ins []model.Instance) {
	pl.events.In() <- &polarisNotification{
		kind:      fullSnapshotNotification,
		instances: copyInstances(ins),
	}
}

// nextNotification returns the next tagged notification in FIFO order.
func (pl *polarisListener) nextNotification() (*polarisNotification, error) {
	for {
		select {
		case <-pl.closeCh:
			logger.Warn("[Registry][Polaris] polaris listener is close")
			return nil, perrors.New("listener stopped")
		case val := <-pl.events.Out():
			notification, ok := val.(*polarisNotification)
			if !ok || notification == nil {
				return nil, perrors.Errorf("invalid polaris notification type %T", val)
			}
			return notification, nil
		}
	}
}

// Next returns next service event once received
func (pl *polarisListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-pl.closeCh:
			logger.Warn("[Registry][Polaris] polaris listener is close")
			return nil, perrors.New("listener stopped")
		default:
		}

		if len(pl.pendingEvents) > 0 {
			event := pl.pendingEvents[0]
			pl.pendingEvents[0] = nil
			pl.pendingEvents = pl.pendingEvents[1:]
			return event, nil
		}

		notification, err := pl.nextNotification()
		if err != nil {
			return nil, err
		}
		action := notification.eventType
		if notification.kind == fullSnapshotNotification {
			action = remoting.EventTypeUpdate
		}
		for _, instance := range notification.instances {
			serviceURL := generateUrl(instance)
			if serviceURL == nil {
				continue
			}
			pl.pendingEvents = append(pl.pendingEvents, &registry.ServiceEvent{
				Action:  action,
				Service: serviceURL,
			})
		}
	}
}

// Close closes this listener
func (pl *polarisListener) Close() {
	// TODO need to add UnWatch in polaris
	close(pl.closeCh)
}

func generateUrl(instance model.Instance) *common.URL {
	if validationError := polarisInstanceURLValidationError(instance); validationError != "" {
		logger.Errorf("[Registry][Polaris] %s, instance=%+v", validationError, instance)
		return nil
	}

	path := instance.GetMetadata()["path"]
	myInterface := instance.GetMetadata()["interface"]
	if len(path) == 0 && len(myInterface) != 0 {
		path = "/" + myInterface
	}
	protocol := instance.GetProtocol()
	urlMap := url.Values{}
	for k, v := range instance.GetMetadata() {
		urlMap.Set(k, v)
	}
	urlMap.Set(constant.PolarisInstanceID, instance.GetId())
	urlMap.Set(constant.PolarisInstanceHealthStatus, fmt.Sprintf("%+v", instance.IsHealthy()))
	urlMap.Set(constant.PolarisInstanceIsolatedStatus, fmt.Sprintf("%+v", instance.IsIsolated()))
	instance.GetCircuitBreakerStatus()
	return common.NewURLWithOptions(
		common.WithIp(instance.GetHost()),
		common.WithPort(strconv.Itoa(int(instance.GetPort()))),
		common.WithProtocol(protocol),
		common.WithParams(urlMap),
		common.WithPath(path),
	)
}

func polarisInstanceURLValidationError(instance model.Instance) string {
	if instance.GetMetadata() == nil {
		return "polaris instance metadata is empty"
	}
	path := instance.GetMetadata()["path"]
	myInterface := instance.GetMetadata()["interface"]
	if len(path) == 0 && len(myInterface) == 0 {
		return "polaris instance metadata does not have both path key and interface key"
	}
	if len(instance.GetProtocol()) == 0 {
		return "polaris instance metadata does not have protocol key"
	}
	return ""
}
