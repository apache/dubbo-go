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
	"dubbo.apache.org/dubbo-go/v3/config_center"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type polarisListener struct {
	watcher   *PolarisServiceWatcher
	listenUrl *common.URL
	events    *gxchan.UnboundedChan
	closeCh   chan struct{}
}

// NewPolarisListener new polaris listener
func NewPolarisListener(url *common.URL) (*polarisListener, error) {
	listener := &polarisListener{
		listenUrl: url,
		events:    gxchan.NewUnboundedChan(32),
		closeCh:   make(chan struct{}),
	}
	return listener, nil
}

// Next returns next service event once received
func (pl *polarisListener) Next() (*registry.ServiceEvent, error) {
	for {
		select {
		case <-pl.closeCh:
			logger.Warnf("polaris listener is close!listenUrl:%+v", pl.listenUrl)
			return nil, perrors.New("listener stopped")
		case val := <-pl.events.Out():
			e, _ := val.(*config_center.ConfigChangeEvent)
			logger.Debugf("got polaris event %s", e)
			instance := e.Value.(model.Instance)
			return &registry.ServiceEvent{Action: e.ConfigType, Service: generateUrl(instance)}, nil
		}
	}
}

// Close closes this listener
func (pl *polarisListener) Close() {
	// TODO need to add UnWatch in polaris
	close(pl.closeCh)
}

func generateUrl(instance model.Instance) *common.URL {
	if instance.GetMetadata() == nil {
		logger.Errorf("polaris instance metadata is empty,instance:%+v", instance)
		return nil
	}
	path := instance.GetMetadata()["path"]
	myInterface := instance.GetMetadata()["interface"]
	if len(path) == 0 && len(myInterface) == 0 {
		logger.Errorf("polaris instance metadata does not have  both path key and interface key,instance:%+v", instance)
		return nil
	}
	if len(path) == 0 && len(myInterface) != 0 {
		path = "/" + myInterface
	}
	protocol := instance.GetProtocol()
	if len(protocol) == 0 {
		logger.Errorf("polaris instance metadata does not have protocol key,instance:%+v", instance)
		return nil
	}
	urlMap := url.Values{}
	for k, v := range instance.GetMetadata() {
		urlMap.Set(k, v)
	}
	return common.NewURLWithOptions(
		common.WithIp(instance.GetHost()),
		common.WithPort(strconv.Itoa(int(instance.GetPort()))),
		common.WithProtocol(protocol),
		common.WithParams(urlMap),
		common.WithPath(path),
	)
}
