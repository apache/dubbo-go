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

package ewatcher

import (
	"fmt"
	"strconv"
	"sync"
)

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/remoting"
	xdsCommon "dubbo.apache.org/dubbo-go/v3/remoting/xds/common"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource"
)

// endPointWatcherCtx is endpoint watching context
type endPointWatcherCtx struct {
	clusterName   string
	interfaceName string
	hostAddr      string

	cancel func()
	/*
		hostAddrListenerMap[hostAddr][serviceUniqueKey] -> registry.NotifyListener
		stores all directory listener, which receives events and refresh invokers
	*/
	hostAddrListenerMap     map[string]map[string]registry.NotifyListener
	hostAddrListenerMapLock *sync.RWMutex
}

func NewEndpointWatcherCtxImpl(
	clusterName,
	hostAddr,
	interfaceName string,
	hostAddrListenerMapLock *sync.RWMutex,
	hostAddrListenerMap map[string]map[string]registry.NotifyListener) EWatcher {
	return &endPointWatcherCtx{
		clusterName:             clusterName,
		hostAddr:                hostAddr,
		interfaceName:           interfaceName,
		hostAddrListenerMapLock: hostAddrListenerMapLock,
		hostAddrListenerMap:     hostAddrListenerMap,
	}
}

func (watcher *endPointWatcherCtx) SetCancelFunction(cancel func()) {
	watcher.cancel = cancel
}

// Handle handles endpoint update event and send to directory to refresh invoker
func (watcher *endPointWatcherCtx) Handle(update resource.EndpointsUpdate, err error) {
	for _, v := range update.Localities {
		for _, e := range v.Endpoints {
			event := generateRegistryEvent(watcher.clusterName, e, watcher.interfaceName)
			watcher.hostAddrListenerMapLock.RLock()
			for _, l := range watcher.hostAddrListenerMap[watcher.hostAddr] {
				// notify all listeners listening this hostAddr
				l.Notify(event)
			}
			watcher.hostAddrListenerMapLock.RUnlock()
		}
	}
}

// Destroy call cancel and send event to listener to remove related invokers of current deleated cluster
func (watcher *endPointWatcherCtx) Destroy() {
	if watcher.cancel != nil {
		watcher.cancel()
	}
	/*
		directory would identify this by EndpointHealthStatusUnhealthy and Location == "*" and none empty clusterID
		and delete related invokers
	*/
	event := generateRegistryEvent(watcher.clusterName, resource.Endpoint{
		HealthStatus: resource.EndpointHealthStatusUnhealthy,
		Address:      constant.MeshAnyAddrMatcher, // Destroy all endpoint of this cluster
	}, watcher.interfaceName)
	watcher.hostAddrListenerMapLock.RLock()
	for _, l := range watcher.hostAddrListenerMap[watcher.hostAddr] {
		// notify all listeners listening this hostAddr
		l.Notify(event)
	}
	watcher.hostAddrListenerMapLock.RUnlock()
}

func generateRegistryEvent(clusterID string, endpoint resource.Endpoint, interfaceName string) *registry.ServiceEvent {
	// todo now only support triple protocol
	url, _ := common.NewURL(fmt.Sprintf("tri://%s/%s", endpoint.Address, interfaceName))
	logger.Infof("[XDS Registry] Get Update event from pilot: interfaceName = %s, addr = %s, healthy = %d\n",
		interfaceName, endpoint.Address, endpoint.HealthStatus)
	cluster := xdsCommon.NewCluster(clusterID)
	// todo const MeshSubsetKey
	url.AddParam(constant.MeshSubsetKey, cluster.Subset)
	url.AddParam(constant.MeshClusterIDKey, clusterID)
	url.AddParam(constant.MeshHostAddrKey, cluster.Addr.String())
	url.AddParam(constant.EndPointWeight, strconv.Itoa(int(endpoint.Weight)))
	if endpoint.HealthStatus == resource.EndpointHealthStatusUnhealthy {
		return &registry.ServiceEvent{
			Action:  remoting.EventTypeDel,
			Service: url,
		}
	}
	return &registry.ServiceEvent{
		Action:  remoting.EventTypeUpdate,
		Service: url,
	}
}

type EWatcher interface {
	Destroy()
	Handle(update resource.EndpointsUpdate, err error)
	SetCancelFunction(cancel func())
}
