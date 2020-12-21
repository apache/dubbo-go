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

package consul

import (
	"strconv"
	"time"
)

import (
	consul "github.com/hashicorp/consul/api"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
)

const (
	registryConnDelay             = 3
	registryDestroyDefaultTimeout = time.Second * 3
)

func init() {
	extension.SetRegistry("consul", newConsulRegistry)
}

// Consul registry wraps the consul client, to
// register and subscribe service.
type consulRegistry struct {
	// Registry url.
	*common.URL

	// Consul client.
	client *consul.Client

	// Done field represents whether
	// consul registry is closed.
	done chan struct{}
}

func newConsulRegistry(url *common.URL) (registry.Registry, error) {
	config := &consul.Config{Address: url.Location}
	client, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}

	r := &consulRegistry{
		URL:    url,
		client: client,
		done:   make(chan struct{}),
	}

	return r, nil
}

// Register register @url
// it delegate the job to register() method
func (r *consulRegistry) Register(url *common.URL) error {
	var err error

	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	if role == common.PROVIDER {
		err = r.register(url)
		if err != nil {
			return err
		}
	}
	return nil
}

// register actually register the @url
func (r *consulRegistry) register(url *common.URL) error {
	service, err := buildService(url)
	if err != nil {
		return err
	}
	return r.client.Agent().ServiceRegister(service)
}

// UnRegister unregister the @url
// it delegate the job to unregister() method
func (r *consulRegistry) UnRegister(url *common.URL) error {
	var err error

	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	if role == common.PROVIDER {
		err = r.unregister(url)
		if err != nil {
			return err
		}
	}
	return nil
}

// unregister actually unregister the @url
func (r *consulRegistry) unregister(url *common.URL) error {
	return r.client.Agent().ServiceDeregister(buildId(url))
}

// Subscribe subscribe the @url with the @notifyListener
func (r *consulRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	if role == common.CONSUMER {
		r.subscribe(url, notifyListener)
	}
	return nil
}

// UnSubscribe is not supported yet
func (r *consulRegistry) UnSubscribe(url *common.URL, notifyListener registry.NotifyListener) error {
	return perrors.New("UnSubscribe not support in consulRegistry")
}

// subscribe actually subscribe the @url
// it loops forever until success
func (r *consulRegistry) subscribe(url *common.URL, notifyListener registry.NotifyListener) {
	for {
		if !r.IsAvailable() {
			logger.Warnf("event listener game over.")
			return
		}

		listener, err := r.getListener(url)
		if err != nil {
			if !r.IsAvailable() {
				logger.Warnf("event listener game over.")
				return
			}
			logger.Warnf("getListener() = err:%v", perrors.WithStack(err))
			time.Sleep(time.Duration(registryConnDelay) * time.Second)
			continue
		}

		for {
			serviceEvent, err := listener.Next()
			if err != nil {
				logger.Warnf("Selector.watch() = error{%v}", perrors.WithStack(err))
				listener.Close()
				return
			}

			logger.Infof("update begin, service event: %v", serviceEvent.String())
			notifyListener.Notify(serviceEvent)
		}
	}
}

func (r *consulRegistry) getListener(url *common.URL) (registry.Listener, error) {
	listener, err := newConsulListener(r.URL, url)
	return listener, err
}

// GetUrl get registry URL of consul registry center
func (r *consulRegistry) GetUrl() *common.URL {
	return r.URL
}

// IsAvailable checks consul registry center whether is available
func (r *consulRegistry) IsAvailable() bool {
	select {
	case <-r.done:
		return false
	default:
		return true
	}
}

// Destroy consul registry center
func (r *consulRegistry) Destroy() {
	if r.URL != nil {
		done := make(chan struct{}, 1)
		go func() {
			defer func() {
				if e := recover(); e != nil {
					logger.Errorf("consulRegistry destory with panic: %v", e)
				}
				done <- struct{}{}
			}()
			if err := r.UnRegister(r.URL); err != nil {
				logger.Errorf("consul registry unregister with err: %s", err.Error())
			}
		}()
		select {
		case <-done:
			logger.Infof("consulRegistry unregister done")
		case <-time.After(registryDestroyDefaultTimeout):
			logger.Errorf("consul unregister timeout")
		}
	}
	close(r.done)
}
