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
	"time"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting/etcdv3"
)

const (
	// Name module name
	Name = "etcdv3"
)

func init() {
	extension.SetRegistry(Name, newETCDV3Registry)
}

type etcdV3Registry struct {
	registry.BaseRegistry
	cltLock        sync.Mutex
	client         *etcdv3.Client
	listenerLock   sync.Mutex
	listener       *etcdv3.EventListener
	dataListener   *dataListener
	configListener *configurationListener
}

// Client get the etcdv3 client
func (r *etcdV3Registry) Client() *etcdv3.Client {
	return r.client
}

//SetClient set the etcdv3 client
func (r *etcdV3Registry) SetClient(client *etcdv3.Client) {
	r.client = client
}

//
func (r *etcdV3Registry) ClientLock() *sync.Mutex {
	return &r.cltLock
}

func newETCDV3Registry(url *common.URL) (registry.Registry, error) {

	timeout, err := time.ParseDuration(url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT))
	if err != nil {
		logger.Errorf("timeout config %v is invalid ,err is %v",
			url.GetParam(constant.REGISTRY_TIMEOUT_KEY, constant.DEFAULT_REG_TIMEOUT), err.Error())
		return nil, perrors.WithMessagef(err, "new etcd registry(address:%+v)", url.Location)
	}

	logger.Infof("etcd address is: %v, timeout is: %s", url.Location, timeout.String())

	r := &etcdV3Registry{}

	r.InitBaseRegistry(url, r)

	if err := etcdv3.ValidateClient(
		r,
		etcdv3.WithName(etcdv3.RegistryETCDV3Client),
		etcdv3.WithTimeout(timeout),
		etcdv3.WithEndpoints(url.Location),
	); err != nil {
		return nil, err
	}
	r.WaitGroup().Add(1) //etcdv3 client start successful, then wg +1

	go etcdv3.HandleClientRestart(r)

	r.InitListeners()

	return r, nil
}

func (r *etcdV3Registry) InitListeners() {
	r.listener = etcdv3.NewEventListener(r.client)
	r.configListener = NewConfigurationListener(r)
	r.dataListener = NewRegistryDataListener(r.configListener)
}

func (r *etcdV3Registry) DoRegister(root string, node string) error {
	return r.client.Create(path.Join(root, node), "")
}

func (r *etcdV3Registry) CloseAndNilClient() {
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	client := r.client
	r.client = nil
	go client.Close()
}

func (r *etcdV3Registry) CloseListener() {
	if r.configListener != nil {
		r.configListener.Close()
	}
}

func (r *etcdV3Registry) CreatePath(k string) error {
	var tmpPath string
	for _, str := range strings.Split(k, "/")[1:] {
		tmpPath = path.Join(tmpPath, "/", str)
		if err := r.client.Create(tmpPath, ""); err != nil {
			return perrors.WithMessagef(err, "create path %s in etcd", tmpPath)
		}
	}

	return nil
}

func (r *etcdV3Registry) DoSubscribe(svc *common.URL) (registry.Listener, error) {

	var (
		configListener *configurationListener
	)

	r.listenerLock.Lock()
	configListener = r.configListener
	r.listenerLock.Unlock()
	if r.listener == nil {
		r.cltLock.Lock()
		client := r.client
		r.cltLock.Unlock()
		if client == nil {
			return nil, perrors.New("etcd client broken")
		}

		// new client & listener
		listener := etcdv3.NewEventListener(r.client)

		r.listenerLock.Lock()
		r.listener = listener
		r.listenerLock.Unlock()
	}

	//register the svc to dataListener
	r.dataListener.AddInterestedURL(svc)
	for _, v := range strings.Split(svc.GetParam(constant.CATEGORY_KEY, constant.DEFAULT_CATEGORY), ",") {
		go r.listener.ListenServiceEvent(fmt.Sprintf("/dubbo/%s/"+v, svc.Service()), r.dataListener)
	}

	return configListener, nil
}
