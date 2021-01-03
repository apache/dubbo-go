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
	"time"
)

import (
	"github.com/dubbogo/getty"
	"github.com/dubbogo/gost/net"
	perrors "github.com/pkg/errors"
	k8s "k8s.io/client-go/kubernetes"
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
	Name         = "kubernetes"
	ConnDelay    = 3
	MaxFailTimes = 15
)

func init() {
	processID = fmt.Sprintf("%d", os.Getpid())
	localIP, _ = gxnet.GetLocalIP()
	extension.SetRegistry(Name, newKubernetesRegistry)
}

type kubernetesRegistry struct {
	registry.BaseRegistry
	cltLock        sync.RWMutex
	client         *kubernetes.Client
	listenerLock   sync.Mutex
	listener       *kubernetes.EventListener
	dataListener   *dataListener
	configListener *configurationListener
}

func (r *kubernetesRegistry) Client() *kubernetes.Client {
	r.cltLock.RLock()
	client := r.client
	r.cltLock.RUnlock()
	return client
}
func (r *kubernetesRegistry) SetClient(client *kubernetes.Client) {
	r.cltLock.Lock()
	r.client = client
	r.cltLock.Unlock()
}

func (r *kubernetesRegistry) CloseAndNilClient() {
	r.client.Close()
	r.client = nil
}

func (r *kubernetesRegistry) CloseListener() {

	r.cltLock.Lock()
	l := r.configListener
	r.cltLock.Unlock()
	if l != nil {
		l.Close()
	}
	r.configListener = nil
}

func (r *kubernetesRegistry) CreatePath(k string) error {
	if err := r.client.Create(k, ""); err != nil {
		return perrors.WithMessagef(err, "create path %s in kubernetes", k)
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

		r.listenerLock.Lock()
		if r.listener == nil {
			// double check
			r.listener = kubernetes.NewEventListener(r.client)
		}
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
		return nil, perrors.WithStack(err)
	}

	r.WaitGroup().Add(1)
	go r.HandleClientRestart()
	r.InitListeners()

	logger.Debugf("the kubernetes registry started")

	return r, nil
}

func newMockKubernetesRegistry(
	url *common.URL,
	namespace string,
	clientGeneratorFunc func() (k8s.Interface, error),
) (registry.Registry, error) {

	var err error

	r := &kubernetesRegistry{}

	r.InitBaseRegistry(url, r)
	r.client, err = kubernetes.NewMockClient(namespace, clientGeneratorFunc)
	if err != nil {
		return nil, perrors.WithMessage(err, "new mock client")
	}
	r.InitListeners()
	return r, nil
}

func (r *kubernetesRegistry) HandleClientRestart() {

	var (
		err       error
		failTimes int
	)

	defer r.WaitGroup().Done()
LOOP:
	for {
		select {
		case <-r.Done():
			logger.Warnf("(KubernetesProviderRegistry)reconnectKubernetes goroutine exit now...")
			break LOOP
			// re-register all services
		case <-r.Client().Done():
			r.Client().Close()
			r.SetClient(nil)

			// try to connect to kubernetes,
			failTimes = 0
			for {
				select {
				case <-r.Done():
					logger.Warnf("(KubernetesProviderRegistry)reconnectKubernetes Registry goroutine exit now...")
					break LOOP
				case <-getty.GetTimeWheel().After(timeSecondDuration(failTimes * ConnDelay)): // avoid connect frequent
				}
				err = kubernetes.ValidateClient(r)
				logger.Infof("Kubernetes ProviderRegistry.validateKubernetesClient = error{%#v}", perrors.WithStack(err))

				if err == nil {
					if r.RestartCallBack() {
						break
					}
				}
				failTimes++
				if MaxFailTimes <= failTimes {
					failTimes = MaxFailTimes
				}
			}
		}
	}
}

func timeSecondDuration(sec int) time.Duration {
	return time.Duration(sec) * time.Second
}
