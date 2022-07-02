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
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

import (
	"github.com/dubbogo/gost/log/logger"

	perrors "github.com/pkg/errors"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

const (
	RegistryConnDelay = 3               // connection delay
	MaxWaitInterval   = 3 * time.Second // max wait interval
)

var (
	processID = ""
	localIP   = ""
)

func init() {
	processID = fmt.Sprintf("%d", os.Getpid())
	localIP = common.GetLocalIp()
}

type createPathFunc func(dubboPath string) error

// FacadeBasedRegistry is the interface of Registry, and it is designed for registry who
// want to inherit BaseRegistry. If there is no special case, you'd better inherit BaseRegistry
// and implement the FacadeBasedRegistry interface instead of directly implementing the
// Registry interface.
//
// CreatePath method create the path in the registry.
//
// DoRegister method actually does the register job.
//
// DoUnregister method does the unregister job.
//
// DoSubscribe method actually subscribes the URL.
//
// DoUnsubscribe method does unsubscribe the URL.
//
// CloseAndNilClient method closes the client and then reset the client in registry to nil
// you should notice that this method will be invoked inside a lock.
// So you should implement this method as light weighted as you can.
//
// CloseListener method closes listeners.
//
// InitListeners method init listeners
type FacadeBasedRegistry interface {
	Registry

	CreatePath(string) error
	DoRegister(string, string) error
	DoUnregister(string, string) error
	DoSubscribe(conf *common.URL) (Listener, error)
	DoUnsubscribe(conf *common.URL) (Listener, error)
	CloseAndNilClient()
	CloseListener()
	InitListeners()
}

// BaseRegistry is a common logic abstract for registry. It implement Registry interface.
type BaseRegistry struct {
	facadeBasedRegistry FacadeBasedRegistry
	*common.URL
	birth      int64          // time of file birth, seconds since Epoch; 0 if unknown
	wg         sync.WaitGroup // wg+done for zk restart
	done       chan struct{}
	registered *sync.Map
	cltLock    sync.RWMutex
}

// InitBaseRegistry for init some local variables and set BaseRegistry's subclass to it
func (r *BaseRegistry) InitBaseRegistry(url *common.URL, facadeRegistry FacadeBasedRegistry) Registry {
	r.URL = url
	r.birth = time.Now().UnixNano()
	r.done = make(chan struct{})
	r.registered = &sync.Map{}
	r.facadeBasedRegistry = facadeRegistry
	return r
}

// GetURL for get registry's url
func (r *BaseRegistry) GetURL() *common.URL {
	return r.URL
}

// Destroy for graceful down
func (r *BaseRegistry) Destroy() {
	// first step close registry's all listeners
	r.facadeBasedRegistry.CloseListener()
	// then close r.done to notify other program who listen to it
	close(r.Done())
	// wait waitgroup done (wait listeners outside close over)
	r.WaitGroup().Wait()

	// close registry client
	r.closeRegisters()
}

// Register implement interface registry to register
func (r *BaseRegistry) Register(url *common.URL) error {
	// if developer define registry port and ip, use it first.
	if ipToRegistry := os.Getenv("DUBBO_IP_TO_REGISTRY"); len(ipToRegistry) > 0 {
		url.Ip = ipToRegistry
	} else {
		url.Ip = common.GetLocalIp()
	}
	if portToRegistry := os.Getenv("DUBBO_PORT_TO_REGISTRY"); len(portToRegistry) > 0 {
		url.Port = portToRegistry
	}
	// todo bug when provider、consumer simultaneous initialization
	if _, ok := r.registered.Load(url.Key()); ok {
		return perrors.Errorf("Service {%s} has been registered", url.Key())
	}

	err := r.register(url)
	if err == nil {
		r.registered.Store(url.Key(), url)

	} else {
		err = perrors.WithMessagef(err, "register(url:%+v)", url)
	}

	return err
}

// UnRegister implement interface registry to unregister
func (r *BaseRegistry) UnRegister(url *common.URL) error {
	if _, ok := r.registered.Load(url.Key()); !ok {
		return perrors.Errorf("Service {%s} has not registered", url.Key())
	}

	err := r.unregister(url)
	if err == nil {
		r.registered.Delete(url.Key())
	} else {
		err = perrors.WithMessagef(err, "unregister(url:%+v)", url)
	}

	return err
}

// service is for getting service path stored in url
func (r *BaseRegistry) service(c *common.URL) string {
	return url.QueryEscape(c.Service())
}

// RestartCallBack for reregister when reconnect
func (r *BaseRegistry) RestartCallBack() bool {
	flag := true
	r.registered.Range(func(key, value interface{}) bool {
		registeredUrl := value.(*common.URL)
		err := r.register(registeredUrl)
		if err != nil {
			flag = false
			logger.Errorf("failed to re-register service :%v, error{%#v}",
				registeredUrl, perrors.WithStack(err))
			return flag
		}

		logger.Infof("success to re-register service :%v", registeredUrl.Key())
		return flag
	})

	if flag {
		r.facadeBasedRegistry.InitListeners()
	}

	return flag
}

// register for register url to registry, include init params
func (r *BaseRegistry) register(c *common.URL) error {
	return r.processURL(c, r.facadeBasedRegistry.DoRegister, r.createPath)
}

// unregister for unregister url to registry, include init params
func (r *BaseRegistry) unregister(c *common.URL) error {
	return r.processURL(c, r.facadeBasedRegistry.DoUnregister, nil)
}

func (r *BaseRegistry) processURL(c *common.URL, f func(string, string) error, cpf createPathFunc) error {
	if f == nil {
		panic(" Must provide a `function(string, string) error` to process URL. ")
	}
	var (
		err        error
		params     url.Values
		rawURL     string
		encodedURL string
		dubboPath  string
	)
	params = url.Values{}

	c.RangeParams(func(key, value string) bool {
		params.Add(key, value)
		return true
	})

	role, _ := strconv.Atoi(c.GetParam(constant.RegistryRoleKey, ""))
	switch role {
	case common.PROVIDER:
		dubboPath, rawURL, err = r.providerRegistry(c, params, cpf)
	case common.CONSUMER:
		dubboPath, rawURL, err = r.consumerRegistry(c, params, cpf)
	default:
		return perrors.Errorf("@c{%v} type is not referencer or provider", c)
	}
	if err != nil {
		return perrors.WithMessagef(err, "@c{%v} registry fail", c)
	}

	encodedURL = url.QueryEscape(rawURL)
	dubboPath = strings.ReplaceAll(dubboPath, "$", "%24")
	err = f(dubboPath, encodedURL)

	if err != nil {
		return perrors.WithMessagef(err, "register Node(path:%s, url:%s)", dubboPath, rawURL)
	}
	return nil
}

// createPath will create dubbo path in register
func (r *BaseRegistry) createPath(dubboPath string) error {
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	return r.facadeBasedRegistry.CreatePath(dubboPath)
}

// providerRegistry for provider role do
func (r *BaseRegistry) providerRegistry(c *common.URL, params url.Values, f createPathFunc) (string, string, error) {
	var (
		dubboPath string
		rawURL    string
		err       error
	)
	if c.Path == "" || len(c.Methods) == 0 {
		return "", "", perrors.Errorf("conf{Path:%s, Methods:%s}", c.Path, c.Methods)
	}
	dubboPath = fmt.Sprintf("/%s/%s/%s", r.URL.GetParam(constant.RegistryGroupKey, "dubbo"), r.service(c), common.DubboNodes[common.PROVIDER])
	if f != nil {
		err = f(dubboPath)
	}
	if err != nil {
		logger.Errorf("facadeBasedRegistry.CreatePath(path{%s}) = error{%#v}", dubboPath, perrors.WithStack(err))
		return "", "", perrors.WithMessagef(err, "facadeBasedRegistry.CreatePath(path:%s)", dubboPath)
	}
	params.Add(constant.AnyhostKey, "true")

	// Dubbo java consumer to start looking for the provider url,because the category does not match,
	// the provider will not find, causing the consumer can not start, so we use consumers.

	if len(c.Methods) != 0 {
		params.Add(constant.MethodsKey, strings.Join(c.Methods, ","))
	}
	logger.Debugf("provider url params:%#v", params)
	var host string
	if len(c.Ip) == 0 {
		host = localIP
	} else {
		host = c.Ip
	}
	host += ":" + c.Port

	// delete empty param key
	for key, val := range params {
		if len(val) > 0 && val[0] == "" {
			params.Del(key)
		}
	}

	s, _ := url.QueryUnescape(params.Encode())
	rawURL = fmt.Sprintf("%s://%s%s?%s", c.Protocol, host, c.Path, s)
	// Print your own registration service providers.
	logger.Debugf("provider path:%s, url:%s", dubboPath, rawURL)
	return dubboPath, rawURL, nil
}

// consumerRegistry for consumer role do
func (r *BaseRegistry) consumerRegistry(c *common.URL, params url.Values, f createPathFunc) (string, string, error) {
	var (
		dubboPath string
		rawURL    string
		err       error
	)
	dubboPath = fmt.Sprintf("/%s/%s/%s", r.URL.GetParam(constant.RegistryGroupKey, "dubbo"), r.service(c), common.DubboNodes[common.CONSUMER])

	if f != nil {
		err = f(dubboPath)
	}

	if err != nil {
		logger.Errorf("facadeBasedRegistry.CreatePath(path{%s}) = error{%v}", dubboPath, perrors.WithStack(err))
		return "", "", perrors.WithStack(err)
	}

	params.Add("protocol", c.Protocol)
	s, _ := url.QueryUnescape(params.Encode())
	rawURL = fmt.Sprintf("consumer://%s%s?%s", localIP, c.Path, s)
	logger.Debugf("consumer path:%s, url:%s", dubboPath, rawURL)
	return dubboPath, rawURL, nil
}

// sleepWait...
func sleepWait(n int) {
	wait := time.Duration((n + 1) * 2e8)
	if wait > MaxWaitInterval {
		wait = MaxWaitInterval
	}
	time.Sleep(wait)
}

// Subscribe :subscribe from registry, event will notify by notifyListener
func (r *BaseRegistry) Subscribe(url *common.URL, notifyListener NotifyListener) error {
	n := 0
	for {
		n++
		if !r.IsAvailable() {
			logger.Warnf("event listener game over.")
			return perrors.New("BaseRegistry is not available.")
		}

		listener, err := r.facadeBasedRegistry.DoSubscribe(url)
		if err != nil {
			if !r.IsAvailable() {
				logger.Warnf("event listener game over.")
				return err
			}
			logger.Warnf("getListener() = err:%v", perrors.WithStack(err))
			time.Sleep(time.Duration(RegistryConnDelay) * time.Second)
			continue
		}

		for {
			if serviceEvent, err := listener.Next(); err != nil {
				logger.Warnf("Selector.watch() = error{%v}", perrors.WithStack(err))
				listener.Close()
				break
			} else {
				logger.Debugf("[Zookeeper Registry] update begin, service event: %v", serviceEvent.String())
				notifyListener.Notify(serviceEvent)
			}
		}
		sleepWait(n)
	}
}

// UnSubscribe URL
func (r *BaseRegistry) UnSubscribe(url *common.URL, notifyListener NotifyListener) error {
	if !r.IsAvailable() {
		logger.Warnf("event listener game over.")
		return perrors.New("BaseRegistry is not available.")
	}

	listener, err := r.facadeBasedRegistry.DoUnsubscribe(url)
	if err != nil {
		if !r.IsAvailable() {
			logger.Warnf("event listener game over.")
			return perrors.New("BaseRegistry is not available.")
		}
		logger.Warnf("getListener() = err:%v", perrors.WithStack(err))
		return perrors.WithStack(err)
	}

	for {
		if serviceEvent, err := listener.Next(); err != nil {
			logger.Warnf("Selector.watch() = error{%v}", perrors.WithStack(err))
			listener.Close()
			break
		} else {
			logger.Debugf("[Zookeeper Registry] update begin, service event: %v", serviceEvent.String())
			notifyListener.Notify(serviceEvent)
		}
	}
	return nil
}

// closeRegisters close and remove registry client and reset services map
func (r *BaseRegistry) closeRegisters() {
	logger.Infof("begin to close provider client")
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	// Close and remove(set to nil) the registry client
	r.facadeBasedRegistry.CloseAndNilClient()
	// reset the services map
	r.registered = nil
}

// IsAvailable judge to is registry not closed by chan r.done
func (r *BaseRegistry) IsAvailable() bool {
	select {
	case <-r.Done():
		return false
	default:
		return true
	}
}

// WaitGroup open for outside add the waitgroup to add some logic before registry destroyed over(graceful down)
func (r *BaseRegistry) WaitGroup() *sync.WaitGroup {
	return &r.wg
}

// Done open for outside to listen the event of registry Destroy() called.
func (r *BaseRegistry) Done() chan struct{} {
	return r.done
}
