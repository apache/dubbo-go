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
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/logger"
)

const (
	// RegistryConnDelay connection delay
	RegistryConnDelay = 3
	// MaxWaitInterval max wait interval
	MaxWaitInterval = 3 * time.Second
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

/*
 * -----------------------------------NOTICE---------------------------------------------
 * If there is no special case, you'd better inherit BaseRegistry and implement the
 * FacadeBasedRegistry interface instead of directly implementing the Registry interface.
 * --------------------------------------------------------------------------------------
 */

/*
 * FacadeBasedRegistry interface is subclass of Registry, and it is designed for registry who want to inherit BaseRegistry.
 * You have to implement the interface to inherit BaseRegistry.
 */
type FacadeBasedRegistry interface {
	Registry

	// CreatePath create the path in the registry
	CreatePath(string) error
	// DoRegister actually do the register job
	DoRegister(string, string) error
	// DoUnregister do the unregister job
	DoUnregister(string, string) error
	// DoSubscribe actually subscribe the URL
	DoSubscribe(conf *common.URL) (Listener, error)
	// DoUnsubscribe does unsubscribe the URL
	DoUnsubscribe(conf *common.URL) (Listener, error)
	// CloseAndNilClient close the client and then reset the client in registry to nil
	// you should notice that this method will be invoked inside a lock.
	// So you should implement this method as light weighted as you can.
	CloseAndNilClient()
	// CloseListener close listeners
	CloseListener()
	// InitListeners init listeners
	InitListeners()
}

// BaseRegistry is a common logic abstract for registry. It implement Registry interface.
type BaseRegistry struct {
	//context             context.Context
	facadeBasedRegistry FacadeBasedRegistry
	*common.URL
	birth    int64          // time of file birth, seconds since Epoch; 0 if unknown
	wg       sync.WaitGroup // wg+done for zk restart
	done     chan struct{}
	cltLock  sync.RWMutex           //ctl lock is a lock for services map
	services map[string]*common.URL // service name + protocol -> service config, for store the service registered
}

// InitBaseRegistry for init some local variables and set BaseRegistry's subclass to it
func (r *BaseRegistry) InitBaseRegistry(url *common.URL, facadeRegistry FacadeBasedRegistry) Registry {
	r.URL = url
	r.birth = time.Now().UnixNano()
	r.done = make(chan struct{})
	r.services = make(map[string]*common.URL)
	r.facadeBasedRegistry = facadeRegistry
	return r
}

// GetUrl for get registry's url
func (r *BaseRegistry) GetUrl() *common.URL {
	return r.URL
}

// Destroy for graceful down
func (r *BaseRegistry) Destroy() {
	//first step close registry's all listeners
	r.facadeBasedRegistry.CloseListener()
	// then close r.done to notify other program who listen to it
	close(r.done)
	// wait waitgroup done (wait listeners outside close over)
	r.wg.Wait()

	//close registry client
	r.closeRegisters()
}

// Register implement interface registry to register
func (r *BaseRegistry) Register(conf *common.URL) error {
	var (
		ok  bool
		err error
	)
	// if developer define registry port and ip, use it first.
	if ipToRegistry := os.Getenv("DUBBO_IP_TO_REGISTRY"); ipToRegistry != "" {
		conf.Ip = ipToRegistry
	}
	if portToRegistry := os.Getenv("DUBBO_PORT_TO_REGISTRY"); portToRegistry != "" {
		conf.Port = portToRegistry
	}
	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
	// Check if the service has been registered
	r.cltLock.Lock()
	_, ok = r.services[conf.Key()]
	r.cltLock.Unlock()
	if ok {
		return perrors.Errorf("Path{%s} has been registered", conf.Key())
	}

	err = r.register(conf)
	if err != nil {
		return perrors.WithMessagef(err, "register(conf:%+v)", conf)
	}

	r.cltLock.Lock()
	r.services[conf.Key()] = conf
	r.cltLock.Unlock()
	logger.Debugf("(%sRegistry)Register(conf{%#v})", common.DubboRole[role], conf)

	return nil
}

// UnRegister implement interface registry to unregister
func (r *BaseRegistry) UnRegister(conf *common.URL) error {
	var (
		ok     bool
		err    error
		oldURL *common.URL
	)

	func() {
		r.cltLock.Lock()
		defer r.cltLock.Unlock()
		oldURL, ok = r.services[conf.Key()]

		if !ok {
			err = perrors.Errorf("Path{%s} has not registered", conf.Key())
		}

		delete(r.services, conf.Key())
	}()

	if err != nil {
		return err
	}

	err = r.unregister(conf)
	if err != nil {
		func() {
			r.cltLock.Lock()
			defer r.cltLock.Unlock()
			r.services[conf.Key()] = oldURL
		}()
		return perrors.WithMessagef(err, "register(conf:%+v)", conf)
	}

	return nil
}

// service is for getting service path stored in url
func (r *BaseRegistry) service(c *common.URL) string {
	return url.QueryEscape(c.Service())
}

// RestartCallBack for reregister when reconnect
func (r *BaseRegistry) RestartCallBack() bool {

	// copy r.services
	services := make([]*common.URL, 0, len(r.services))
	for _, confIf := range r.services {
		services = append(services, confIf)
	}

	flag := true
	for _, confIf := range services {
		err := r.register(confIf)
		if err != nil {
			logger.Errorf("(ZkProviderRegistry)register(conf{%#v}) = error{%#v}",
				confIf, perrors.WithStack(err))
			flag = false
			break
		}
		logger.Infof("success to re-register service :%v", confIf.Key())
	}

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
		err error
		//revision   string
		params     url.Values
		rawURL     string
		encodedURL string
		dubboPath  string
		//conf       config.URL
	)
	params = url.Values{}

	c.RangeParams(func(key, value string) bool {
		params.Add(key, value)
		return true
	})

	params.Add("pid", processID)
	params.Add("ip", localIP)
	//params.Add("timeout", fmt.Sprintf("%d", int64(r.Timeout)/1e6))

	role, _ := strconv.Atoi(r.URL.GetParam(constant.ROLE_KEY, ""))
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
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", r.service(c), common.DubboNodes[common.PROVIDER])
	if f != nil {
		err = f(dubboPath)
	}
	if err != nil {
		logger.Errorf("facadeBasedRegistry.CreatePath(path{%s}) = error{%#v}", dubboPath, perrors.WithStack(err))
		return "", "", perrors.WithMessagef(err, "facadeBasedRegistry.CreatePath(path:%s)", dubboPath)
	}
	params.Add(constant.ANYHOST_KEY, "true")

	// Dubbo java consumer to start looking for the provider url,because the category does not match,
	// the provider will not find, causing the consumer can not start, so we use consumers.

	if len(c.Methods) != 0 {
		params.Add(constant.METHODS_KEY, strings.Join(c.Methods, ","))
	}
	logger.Debugf("provider url params:%#v", params)
	var host string
	if c.Ip == "" {
		host = localIP
	} else {
		host = c.Ip
	}
	host += ":" + c.Port

	//delete empty param key
	for key, val := range params {
		if len(val) > 0 && val[0] == "" {
			params.Del(key)
		}
	}

	s, _ := url.QueryUnescape(params.Encode())
	rawURL = fmt.Sprintf("%s://%s%s?%s", c.Protocol, host, c.Path, s)

	// Print your own registration service providers.
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", r.service(c), (common.RoleType(common.PROVIDER)).String())
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
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", r.service(c), common.DubboNodes[common.CONSUMER])

	if f != nil {
		err = f(dubboPath)
	}
	if err != nil {
		logger.Errorf("facadeBasedRegistry.CreatePath(path{%s}) = error{%v}", dubboPath, perrors.WithStack(err))
		return "", "", perrors.WithStack(err)
	}
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", r.service(c), common.DubboNodes[common.PROVIDER])

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
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", r.service(c), (common.RoleType(common.CONSUMER)).String())

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
				logger.Infof("update begin, service event: %v", serviceEvent.String())
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
			logger.Infof("update begin, service event: %v", serviceEvent.String())
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
	r.services = nil
}

// IsAvailable judge to is registry not closed by chan r.done
func (r *BaseRegistry) IsAvailable() bool {
	select {
	case <-r.done:
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
