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

package etcdv3

import (
	"fmt"
	"path"
	"strings"
	"sync"
)

import (
	gxetcd "github.com/dubbogo/gost/database/kv/etcd/v3"
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting/etcdv3"
)

const (
	Name = "etcdv3"
)

func init() {
	extension.SetRegistry(Name, newETCDV3Registry)
}

type etcdV3Registry struct {
	registry.BaseRegistry
	cltLock      sync.Mutex
	client       *gxetcd.Client
	listenerLock sync.RWMutex
	listener     *etcdv3.EventListener
	dataListener *dataListener
}

// Client gets the etcdv3 client
func (r *etcdV3Registry) Client() *gxetcd.Client {
	return r.client
}

// SetClient sets the etcdv3 client
func (r *etcdV3Registry) SetClient(client *gxetcd.Client) {
	r.client = client
}

// ClientLock returns lock for client
func (r *etcdV3Registry) ClientLock() *sync.Mutex {
	return &r.cltLock
}

func newETCDV3Registry(url *common.URL) (registry.Registry, error) {
	timeout := url.GetParamDuration(constant.RegistryTimeoutKey, constant.DefaultRegTimeout)

	logger.Infof("[Registry][Etcdv3] etcd address=%v timeout=%s", url.Location, timeout.String())

	r := &etcdV3Registry{}

	r.InitBaseRegistry(url, r)

	if err := etcdv3.ValidateClient(
		r,
		gxetcd.WithName(gxetcd.RegistryETCDV3Client),
		gxetcd.WithTimeout(timeout),
		gxetcd.WithEndpoints(strings.Split(url.Location, ",")...),
	); err != nil {
		return nil, err
	}

	r.handleClientRestart()
	r.InitListeners()

	return r, nil
}

// InitListeners init listeners of etcd registry center
func (r *etcdV3Registry) InitListeners() {
	r.listener = etcdv3.NewEventListener(r.client)
	r.dataListener = NewRegistryDataListener()
}

// DoRegister actually do the register job in the registry center of etcd
// for lease
func (r *etcdV3Registry) DoRegister(root string, node string) error {
	return r.client.RegisterTemp(path.Join(root, node), "")
}

// DoUnregister actually unregister the node from the registry center of etcd.
func (r *etcdV3Registry) DoUnregister(root string, node string) error {
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	if !r.client.Valid() {
		return perrors.Errorf("etcd client is not valid.")
	}
	return r.client.Delete(path.Join(root, node))
}

// CloseAndNilClient closes listeners and clear client
func (r *etcdV3Registry) CloseAndNilClient() {
	r.client.Close()
	r.client = nil
}

// CloseListener closes listeners
func (r *etcdV3Registry) CloseListener() {
	if r.dataListener != nil {
		r.dataListener.Close()
	}
}

// CreatePath create the path in the registry center of etcd
func (r *etcdV3Registry) CreatePath(k string) error {
	var tmpPath string
	for _, str := range strings.Split(k, "/")[1:] {
		tmpPath = path.Join(tmpPath, "/", str)
		if err := r.client.Put(tmpPath, ""); err != nil {
			return perrors.WithMessagef(err, "create path %s in etcd", tmpPath)
		}
	}

	return nil
}

// DoSubscribe actually subscribe the provider URL
func (r *etcdV3Registry) DoSubscribe(conf *common.URL) (registry.Listener, error) {
	return r.getListener(conf)
}

func (r *etcdV3Registry) DoUnsubscribe(conf *common.URL) (registry.Listener, error) {
	return r.getCloseListener(conf)
}

func (r *etcdV3Registry) getListener(conf *common.URL) (*configurationListener, error) {
	var etcdListener *configurationListener
	dataListener := r.dataListener

	dataListener.mutex.Lock()
	defer dataListener.mutex.Unlock()

	if dataListener.subscribed[conf.ServiceKey()] != nil {
		etcdListener, _ = r.dataListener.subscribed[conf.ServiceKey()].(*configurationListener)
		if etcdListener != nil {
			if etcdListener.isClosed {
				return nil, perrors.New("configListener already been closed")
			}
			return etcdListener, nil
		}
	}

	etcdListener = NewConfigurationListener(r)
	if r.listener == nil {
		r.cltLock.Lock()
		client := r.client
		r.cltLock.Unlock()
		if client == nil {
			return nil, perrors.New("etcd client broken")
		}
		r.listenerLock.Lock()
		r.listener = etcdv3.NewEventListener(r.client)
		r.listenerLock.Unlock()
	}

	r.dataListener.SubscribeURL(conf, etcdListener)
	go r.listener.ListenServiceEvent(
		fmt.Sprintf("/dubbo/%s/"+constant.DefaultCategory, conf.Service()),
		r.dataListener,
	)

	return etcdListener, nil
}

func (r *etcdV3Registry) getCloseListener(conf *common.URL) (*configurationListener, error) {
	var etcdListener *configurationListener
	r.dataListener.mutex.Lock()
	configListener := r.dataListener.subscribed[conf.ServiceKey()]
	if configListener != nil {
		etcdListener, _ = configListener.(*configurationListener)
		if etcdListener != nil && etcdListener.isClosed {
			r.dataListener.mutex.Unlock()
			return nil, perrors.New(fmt.Sprintf("configListener for service %s has already been closed", conf.ServiceKey()))
		}
	}

	if configListener = r.dataListener.UnSubscribeURL(conf); configListener != nil {
		switch v := configListener.(type) {
		case *configurationListener:
			if v != nil {
				etcdListener = v
			}
		}
	}
	r.dataListener.mutex.Unlock()

	if r.listener == nil {
		return nil, perrors.New("etcd event listener is null, can not close.")
	}

	return etcdListener, nil
}

// LoadSubscribeInstances load subscribe instance
func (r *etcdV3Registry) LoadSubscribeInstances(_ *common.URL, _ registry.NotifyListener) error {
	return nil
}

func (r *etcdV3Registry) handleClientRestart() {
	r.WaitGroup().Add(1)
	go etcdv3.HandleClientRestart(r)
}
