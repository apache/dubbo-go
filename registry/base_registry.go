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
	"context"
	"fmt"
	"net/url"
	"os"
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
	localIP, _ = gxnet.GetLocalIP()
}

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
	CreatePath(string) error
	DoRegister(string, string) error
	DoSubscribe(conf *common.URL) (Listener, error)
	CloseAndNilClient()
	CloseListener()
	InitListeners()
}

/*
 * BaseRegistry is a common logic abstract for registry. It implement Registry interface.
 */
type BaseRegistry struct {
	context             context.Context
	facadeBasedRegistry FacadeBasedRegistry
	*common.URL
	birth    int64          // time of file birth, seconds since Epoch; 0 if unknown
	wg       sync.WaitGroup // wg+done for zk restart
	done     chan struct{}
	cltLock  sync.Mutex            //ctl lock is a lock for services map
	services map[string]common.URL // service name + protocol -> service config, for store the service registered
}

func (r *BaseRegistry) InitBaseRegistry(url *common.URL, facadeRegistry FacadeBasedRegistry) Registry {
	r.URL = url
	r.birth = time.Now().UnixNano()
	r.done = make(chan struct{})
	r.services = make(map[string]common.URL)
	r.facadeBasedRegistry = facadeRegistry
	return r
}

func (r *BaseRegistry) GetUrl() common.URL {
	return *r.URL
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

func (r *BaseRegistry) Register(conf common.URL) error {
	var (
		ok  bool
		err error
	)
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

// service is for getting service path stored in url
func (r *BaseRegistry) service(c common.URL) string {
	return url.QueryEscape(c.Service())
}

func (r *BaseRegistry) RestartCallBack() bool {

	// copy r.services
	services := []common.URL{}
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
	r.facadeBasedRegistry.InitListeners()

	return flag
}

func (r *BaseRegistry) register(c common.URL) error {
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
		dubboPath, rawURL, err = r.providerRegistry(c, params)
	case common.CONSUMER:
		dubboPath, rawURL, err = r.consumerRegistry(c, params)
	default:
		return perrors.Errorf("@c{%v} type is not referencer or provider", c)
	}
	encodedURL = url.QueryEscape(rawURL)
	dubboPath = strings.ReplaceAll(dubboPath, "$", "%24")
	err = r.facadeBasedRegistry.DoRegister(dubboPath, encodedURL)

	if err != nil {
		return perrors.WithMessagef(err, "register Node(path:%s, url:%s)", dubboPath, rawURL)
	}
	return nil
}

func (r *BaseRegistry) providerRegistry(c common.URL, params url.Values) (string, string, error) {
	var (
		dubboPath string
		rawURL    string
		err       error
	)
	if c.Path == "" || len(c.Methods) == 0 {
		return "", "", perrors.Errorf("conf{Path:%s, Methods:%s}", c.Path, c.Methods)
	}
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", r.service(c), common.DubboNodes[common.PROVIDER])
	r.cltLock.Lock()
	err = r.facadeBasedRegistry.CreatePath(dubboPath)
	r.cltLock.Unlock()
	if err != nil {
		logger.Errorf("facadeBasedRegistry.CreatePath(path{%s}) = error{%#v}", dubboPath, perrors.WithStack(err))
		return "", "", perrors.WithMessagef(err, "facadeBasedRegistry.CreatePath(path:%s)", dubboPath)
	}
	params.Add("anyhost", "true")

	// Dubbo java consumer to start looking for the provider url,because the category does not match,
	// the provider will not find, causing the consumer can not start, so we use consumers.
	// DubboRole               = [...]string{"consumer", "", "", "provider"}
	// params.Add("category", (RoleType(PROVIDER)).Role())
	params.Add("category", (common.RoleType(common.PROVIDER)).String())
	params.Add("dubbo", "dubbo-provider-golang-"+constant.Version)

	params.Add("side", (common.RoleType(common.PROVIDER)).Role())

	if len(c.Methods) == 0 {
		params.Add("methods", strings.Join(c.Methods, ","))
	}
	logger.Debugf("provider url params:%#v", params)
	var host string
	if c.Ip == "" {
		host = localIP + ":" + c.Port
	} else {
		host = c.Ip + ":" + c.Port
	}

	rawURL = fmt.Sprintf("%s://%s%s?%s", c.Protocol, host, c.Path, params.Encode())
	// Print your own registration service providers.
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", r.service(c), (common.RoleType(common.PROVIDER)).String())
	logger.Debugf("provider path:%s, url:%s", dubboPath, rawURL)
	return dubboPath, rawURL, nil
}

func (r *BaseRegistry) consumerRegistry(c common.URL, params url.Values) (string, string, error) {
	var (
		dubboPath string
		rawURL    string
		err       error
	)
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", r.service(c), common.DubboNodes[common.CONSUMER])
	r.cltLock.Lock()
	err = r.facadeBasedRegistry.CreatePath(dubboPath)
	r.cltLock.Unlock()
	if err != nil {
		logger.Errorf("facadeBasedRegistry.CreatePath(path{%s}) = error{%v}", dubboPath, perrors.WithStack(err))
		return "", "", perrors.WithStack(err)
	}
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", r.service(c), common.DubboNodes[common.PROVIDER])
	r.cltLock.Lock()
	err = r.facadeBasedRegistry.CreatePath(dubboPath)
	r.cltLock.Unlock()
	if err != nil {
		logger.Errorf("facadeBasedRegistry.CreatePath(path{%s}) = error{%v}", dubboPath, perrors.WithStack(err))
		return "", "", perrors.WithStack(err)
	}

	params.Add("protocol", c.Protocol)
	params.Add("category", (common.RoleType(common.CONSUMER)).String())
	params.Add("dubbo", "dubbogo-consumer-"+constant.Version)

	rawURL = fmt.Sprintf("consumer://%s%s?%s", localIP, c.Path, params.Encode())
	dubboPath = fmt.Sprintf("/dubbo/%s/%s", r.service(c), (common.RoleType(common.CONSUMER)).String())

	logger.Debugf("consumer path:%s, url:%s", dubboPath, rawURL)
	return dubboPath, rawURL, nil
}

func sleepWait(n int) {
	wait := time.Duration((n + 1) * 2e8)
	if wait > MaxWaitInterval {
		wait = MaxWaitInterval
	}
	time.Sleep(wait)
}

//Subscribe :subscribe from registry, event will notify by notifyListener
func (r *BaseRegistry) Subscribe(url *common.URL, notifyListener NotifyListener) {
	n := 0
	for {
		n++
		if !r.IsAvailable() {
			logger.Warnf("event listener game over.")
			return
		}

		listener, err := r.facadeBasedRegistry.DoSubscribe(url)
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
				break
			} else {
				logger.Infof("update begin, service event: %v", serviceEvent.String())
				notifyListener.Notify(serviceEvent)
			}

		}
		sleepWait(n)
	}
}

func (r *BaseRegistry) closeRegisters() {
	r.cltLock.Lock()
	defer r.cltLock.Unlock()
	logger.Infof("begin to close provider client")
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
