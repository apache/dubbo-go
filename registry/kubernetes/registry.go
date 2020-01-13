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
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	gxnet "github.com/dubbogo/gost/net"
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
	Name              = "kubernetes"
	RegistryConnDelay = 3
)

func init() {
	processID = fmt.Sprintf("%d", os.Getpid())
	localIP, _ = gxnet.GetLocalIP()
	extension.SetRegistry(Name, newKubernetesRegistry)
}

type kubernetesRegistry struct {
	*common.URL
	birth int64 // time of file birth, seconds since Epoch; 0 if unknown

	cltLock  sync.Mutex
	client   *kubernetes.Client
	services map[string]common.URL // service name + protocol -> service config

	listenerLock   sync.Mutex
	listener       *kubernetes.EventListener
	dataListener   *dataListener
	configListener *configurationListener

	wg   sync.WaitGroup // wg+done for kubernetes client restart
	done chan struct{}
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
func (r *kubernetesRegistry) WaitGroup() *sync.WaitGroup {
	return &r.wg
}
func (r *kubernetesRegistry) GetDone() chan struct{} {
	return r.done
}
func (r *kubernetesRegistry) RestartCallBack() bool {

	services := []common.URL{}
	for _, confIf := range r.services {
		services = append(services, confIf)
	}

	for _, confIf := range services {
		err := r.Register(confIf)
		if err != nil {
			logger.Errorf("(kubernetesProviderRegistry)register(conf{%#v}) = error{%#v}",
				confIf, perrors.WithStack(err))
			return false
		}
		logger.Infof("success to re-register service :%v", confIf.Key())
	}
	return true
}

func newKubernetesRegistry(url *common.URL) (registry.Registry, error) {

	r := &kubernetesRegistry{
		URL:      url,
		birth:    time.Now().UnixNano(),
		done:     make(chan struct{}),
		services: make(map[string]common.URL),
	}

	if err := kubernetes.ValidateClient(r); err != nil {
		return nil, err
	}

	r.wg.Add(1)
	go kubernetes.HandleClientRestart(r)

	r.listener = kubernetes.NewEventListener(r.client)
	r.configListener = NewConfigurationListener(r)
	r.dataListener = NewRegistryDataListener(r.configListener)

	return r, nil
}

func (r *kubernetesRegistry) GetUrl() common.URL {
	return *r.URL
}

func (r *kubernetesRegistry) IsAvailable() bool {

	select {
	case <-r.done:
		return false
	default:
		return true
	}
}

func (r *kubernetesRegistry) Destroy() {

	if r.configListener != nil {
		r.configListener.Close()
	}
	r.stop()
}

func (r *kubernetesRegistry) stop() {

	close(r.done)

	// close current client
	r.client.Close()

	r.cltLock.Lock()
	r.client = nil
	r.services = nil
	r.cltLock.Unlock()
}

func (r *kubernetesRegistry) Register(svc common.URL) error {

	role, err := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	if err != nil {
		return perrors.WithMessage(err, "get registry role")
	}

	r.cltLock.Lock()
	if _, ok := r.services[svc.Key()]; ok {
		r.cltLock.Unlock()
		return perrors.New(fmt.Sprintf("Path{%s} has been registered", svc.Path))
	}
	r.cltLock.Unlock()

	switch role {
	case common.PROVIDER:
		logger.Debugf("(provider register )Register(conf{%#v})", svc)
		if err := r.registerProvider(svc); err != nil {
			return perrors.WithMessage(err, "register provider")
		}
	case common.CONSUMER:
		logger.Debugf("(consumer register )Register(conf{%#v})", svc)
		if err := r.registerConsumer(svc); err != nil {
			return perrors.WithMessage(err, "register consumer")
		}
	default:
		return perrors.New(fmt.Sprintf("unknown role %d", role))
	}

	r.cltLock.Lock()
	r.services[svc.Key()] = svc
	r.cltLock.Unlock()
	return nil
}

func (r *kubernetesRegistry) createDirIfNotExist(k string) error {

	var tmpPath string
	for _, str := range strings.Split(k, "/")[1:] {
		tmpPath = path.Join(tmpPath, "/", str)
		if err := r.client.Create(tmpPath, ""); err != nil {
			return perrors.WithMessagef(err, "create path %s in kubernetes", tmpPath)
		}
	}

	return nil
}

func (r *kubernetesRegistry) registerConsumer(svc common.URL) error {

	consumersNode := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), common.DubboNodes[common.CONSUMER])
	if err := r.createDirIfNotExist(consumersNode); err != nil {
		logger.Errorf("kubernetes client create path %s: %v", consumersNode, err)
		return perrors.WithMessage(err, "kubernetes create consumer nodes")
	}
	providersNode := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), common.DubboNodes[common.PROVIDER])
	if err := r.createDirIfNotExist(providersNode); err != nil {
		return perrors.WithMessage(err, "create provider node")
	}

	params := url.Values{}

	params.Add("protocol", svc.Protocol)

	params.Add("category", (common.RoleType(common.CONSUMER)).String())
	params.Add("dubbo", "dubbogo-consumer-"+constant.Version)

	encodedURL := url.QueryEscape(fmt.Sprintf("consumer://%s%s?%s", localIP, svc.Path, params.Encode()))
	dubboPath := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), (common.RoleType(common.CONSUMER)).String())
	if err := r.client.Create(path.Join(dubboPath, encodedURL), ""); err != nil {
		return perrors.WithMessagef(err, "create k/v in kubernetes (path:%s, url:%s)", dubboPath, encodedURL)
	}

	return nil
}

func (r *kubernetesRegistry) registerProvider(svc common.URL) error {

	if len(svc.Path) == 0 || len(svc.Methods) == 0 {
		return perrors.New(fmt.Sprintf("service path %s or service method %s", svc.Path, svc.Methods))
	}

	var (
		urlPath    string
		encodedURL string
		dubboPath  string
	)

	providersNode := fmt.Sprintf("/dubbo/%s/%s", svc.Service(), common.DubboNodes[common.PROVIDER])
	if err := r.createDirIfNotExist(providersNode); err != nil {
		return perrors.WithMessage(err, "create provider node")
	}

	params := url.Values{}

	svc.RangeParams(func(key, value string) bool {
		params[key] = []string{value}
		return true
	})
	params.Add("pid", processID)
	params.Add("ip", localIP)
	params.Add("anyhost", "true")
	params.Add("category", (common.RoleType(common.PROVIDER)).String())
	params.Add("dubbo", "dubbo-provider-golang-"+constant.Version)
	params.Add("side", (common.RoleType(common.PROVIDER)).Role())

	logger.Debugf("provider url params:%#v", params)
	var host string
	if len(svc.Ip) == 0 {
		host = localIP + ":" + svc.Port
	} else {
		host = svc.Ip + ":" + svc.Port
	}

	urlPath = svc.Path

	encodedURL = url.QueryEscape(fmt.Sprintf("%s://%s%s?%s", svc.Protocol, host, urlPath, params.Encode()))
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", svc.Service(), (common.RoleType(common.PROVIDER)).String())

	if err := r.client.Create(path.Join(dubboPath, encodedURL), ""); err != nil {
		return perrors.WithMessagef(err, "create k/v in kubernetes (path:%s, url:%s)", dubboPath, encodedURL)
	}

	return nil
}

func (r *kubernetesRegistry) subscribe(svc *common.URL) (registry.Listener, error) {

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
		// NOTICE:
		// double-check the listener
		// if r.listener already be assigned, discard the new value
		if r.listener == nil {
			r.listener = listener
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

//subscribe from registry
func (r *kubernetesRegistry) Subscribe(url *common.URL, notifyListener registry.NotifyListener) {
	for {
		if !r.IsAvailable() {
			logger.Warnf("event listener game over.")
			return
		}

		listener, err := r.subscribe(url)
		if err != nil {
			if !r.IsAvailable() {
				logger.Warnf("event listener game over.")
				return
			}
			logger.Warnf("getListener() = err:%v", perrors.WithStack(err))
			time.Sleep(time.Duration(RegistryConnDelay) * time.Second)
			continue
		}

		for {
			if serviceEvent, err := listener.Next(); err != nil {
				logger.Warnf("Selector.watch() = error{%v}", perrors.WithStack(err))
				listener.Close()
				return
			} else {
				logger.Infof("update begin, service event: %v", serviceEvent.String())
				notifyListener.Notify(serviceEvent)
			}

		}

	}
}
