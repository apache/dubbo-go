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
	"time"
)

import (
	"go.uber.org/atomic"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/logger"
)

type MockRegistry struct {
	listener  *listener
	destroyed *atomic.Bool
}

func NewMockRegistry(url *common.URL) (Registry, error) {
	registry := &MockRegistry{
		destroyed: atomic.NewBool(false),
	}
	listener := &listener{count: 0, registry: registry, listenChan: make(chan *ServiceEvent)}
	registry.listener = listener
	return registry, nil
}
func (*MockRegistry) Register(url common.URL) error {
	return nil
}

func (r *MockRegistry) Destroy() {
	if r.destroyed.CAS(false, true) {
	}
}
func (r *MockRegistry) IsAvailable() bool {
	return !r.destroyed.Load()
}
func (r *MockRegistry) GetUrl() common.URL {
	return common.URL{}
}

func (r *MockRegistry) subscribe(*common.URL) (Listener, error) {
	return r.listener, nil
}
func (r *MockRegistry) Subscribe(url *common.URL, notifyListener NotifyListener) {
	go func() {
		for {
			if !r.IsAvailable() {
				logger.Warnf("event listener game over.")
				time.Sleep(time.Duration(3) * time.Second)
				return
			}

			listener, err := r.subscribe(url)
			if err != nil {
				if !r.IsAvailable() {
					logger.Warnf("event listener game over.")
					return
				}
				time.Sleep(time.Duration(3) * time.Second)
				continue
			}

			for {
				if serviceEvent, err := listener.Next(); err != nil {
					listener.Close()
					time.Sleep(time.Duration(3) * time.Second)
					return
				} else {
					logger.Infof("update begin, service event: %v", serviceEvent.String())
					notifyListener.Notify(serviceEvent)
				}

			}

		}
	}()
}

type listener struct {
	count      int64
	registry   *MockRegistry
	listenChan chan *ServiceEvent
}

func (l *listener) Next() (*ServiceEvent, error) {
	select {
	case e := <-l.listenChan:
		return e, nil
	}
}

func (*listener) Close() {

}

func (r *MockRegistry) MockEvent(event *ServiceEvent) {
	r.listener.listenChan <- event
}
