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

package registry

import (
	"fmt"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	"go.uber.org/atomic"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

// MockRegistry is used as mock registry
type MockRegistry struct {
	listener   *listener
	destroyed  *atomic.Bool
	allAddress chan []*ServiceEvent
}

// NewMockRegistry creates a mock registry
func NewMockRegistry(url *common.URL) (Registry, error) {
	registry := &MockRegistry{
		destroyed:  atomic.NewBool(false),
		allAddress: make(chan []*ServiceEvent),
	}
	listener := &listener{count: 0, registry: registry, listenChan: make(chan *ServiceEvent)}
	registry.listener = listener
	return registry, nil
}

// Register is used as a mock registry
func (*MockRegistry) Register(url *common.URL) error {
	return nil
}

// nolint
func (r *MockRegistry) UnRegister(conf *common.URL) error {
	return nil
}

// nolint
func (r *MockRegistry) Destroy() {
	if r.destroyed.CAS(false, true) {
	}
}

// IsAvailable is use for determine a mock registry available
func (r *MockRegistry) IsAvailable() bool {
	return !r.destroyed.Load()
}

// nolint
func (r *MockRegistry) GetURL() *common.URL {
	return nil
}

func (r *MockRegistry) subscribe(*common.URL) (Listener, error) {
	return r.listener, nil
}

// nolint
func (r *MockRegistry) Subscribe(url *common.URL, notifyListener NotifyListener) error {
	go func() {
		for {
			t, listener := r.checkLoopSubscribe(url)
			if t == 0 {
				continue
			} else if t == -1 {
				return
			}
			for {
				serviceEvent, err := listener.Next()
				if err != nil {
					listener.Close()
					time.Sleep(time.Duration(3) * time.Second)
					return
				}

				logger.Infof("[Mock Registry] update begin, service event: %v", serviceEvent.String())
				notifyListener.Notify(serviceEvent)
			}
		}
	}()
	go func() {
		for {
			t, _ := r.checkLoopSubscribe(url)
			if t == 0 {
				continue
			} else if t == -1 {
				return
			}

			for {
				select {
				case e := <-r.allAddress:
					notifyListener.NotifyAll(e, func() {
						fmt.Print("notify all ok")
					})
					break
				}
			}
		}
	}()
	return nil
}

// UnSubscribe :
func (r *MockRegistry) UnSubscribe(url *common.URL, notifyListener NotifyListener) error {
	return nil
}

type listener struct {
	count      int64
	registry   *MockRegistry
	listenChan chan *ServiceEvent
}

func (l *listener) Next() (*ServiceEvent, error) {
	return <-l.listenChan, nil
}

func (*listener) Close() {
}

// nolint
func (r *MockRegistry) MockEvent(event *ServiceEvent) {
	r.listener.listenChan <- event
}

// nolint
func (r *MockRegistry) MockEvents(events []*ServiceEvent) {
	r.allAddress <- events
}

func (r *MockRegistry) checkLoopSubscribe(url *common.URL) (int, Listener) {
	if !r.IsAvailable() {
		logger.Warnf("event listener game over.")
		time.Sleep(time.Duration(3) * time.Second)
		return -1, nil
	}

	listener, err := r.subscribe(url)
	if err != nil {
		if !r.IsAvailable() {
			logger.Warnf("event listener game over.")
			return -1, nil
		}
		time.Sleep(time.Duration(3) * time.Second)
		return 0, nil
	}
	return 1, listener
}
