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

package kubernetes

import (
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
)

import (
	"github.com/dubbogo/gost/net"
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/common/logger"
	"github.com/apache/dubbo-go/registry"
	"github.com/apache/dubbo-go/remoting/kubernetes"
)

var (
	processID = ""
	localIP   = ""
)

const (
	Name = "kubernetes"
)

func init() {
	processID = fmt.Sprintf("%d", os.Getpid())
	localIP, _ = gxnet.GetLocalIP()
	extension.SetRegistry(Name, newKubernetesRegistry)
}

type kubernetesRegistry struct {
	registry.BaseRegistry
	cltLock        sync.Mutex
	client         *kubernetes.Client
	listenerLock   sync.Mutex
	listener       *kubernetes.EventListener
	dataListener   *dataListener
	configListener *configurationListener
}

func (r *kubernetesRegistry) Client() *kubernetes.Client {
	return r.client
}
func (r *kubernetesRegistry) SetClient(client *kubernetes.Client) {
	r.client = client
}
func (r *kubernetesRegistry) ClientLock() *sync.Mutex {
	return &r.cltLock
}

func (r *kubernetesRegistry) CloseAndNilClient() {
	r.client.Close()
	r.client = nil
}

func (r *kubernetesRegistry) CloseListener() {
	if r.configListener != nil {
		r.configListener.Close()
	}
}

func (r *kubernetesRegistry) CreatePath(k string) error {
	var tmpPath string
	for _, str := range strings.Split(k, "/")[1:] {
		tmpPath = path.Join(tmpPath, "/", str)
		if err := r.client.Create(tmpPath, ""); err != nil {
			return perrors.WithMessagef(err, "create path %s in kubernetes", tmpPath)
		}
	}

	return nil
}

func (r *kubernetesRegistry) DoRegister(root string, node string) error {
	return r.client.Create(path.Join(root, node), "")
}

func (r *kubernetesRegistry) DoSubscribe(svc *common.URL) (registry.Listener, error) {

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
			return nil, perrors.New("kubernetes client broken")
		}

		// new client & listener
		listener := kubernetes.NewEventListener(r.client)

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

func (r *kubernetesRegistry) InitListeners() {
	r.listener = kubernetes.NewEventListener(r.client)
	r.configListener = NewConfigurationListener(r)
	r.dataListener = NewRegistryDataListener(r.configListener)
}

func newKubernetesRegistry(url *common.URL) (registry.Registry, error) {

	// actually, kubernetes use in-cluster config,
	r := &kubernetesRegistry{}

	r.InitBaseRegistry(url, r)

	if err := kubernetes.ValidateClient(r); err != nil {
		return nil, err
	}

	r.WaitGroup().Add(1)
	go kubernetes.HandleClientRestart(r)
	r.InitListeners()

	logger.Debugf("the kubernetes registry started")

	return r, nil
}
