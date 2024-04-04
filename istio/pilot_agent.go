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

package istio

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/istio/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/istio/channel"
	"dubbo.apache.org/dubbo-go/v3/istio/protocol"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"github.com/dubbogo/gost/log/logger"
	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
)

var (
	EnableDubboMesh bool
	pilotAgent      XdsAgent
	pilotAgentMutex sync.Once
	pilotAgentErr   error
)

const (
	pilotAgentWaitTimeout = 10 * time.Second
)

type PilotAgent struct {
	bootstrapInfo    *bootstrap.BootstrapInfo
	sdsClientChannel *channel.SdsClientChannel
	xdsClientChannel *channel.XdsClientChannel
	cdsProtocol      *protocol.CdsProtocol
	edsProtocol      *protocol.EdsProtocol
	ldsProtocol      *protocol.LdsProtocol
	rdsProtocol      *protocol.RdsProtocol
	secretProtocol   *protocol.SecretProtocol
	stopChan         chan struct{}
	updateChan       chan resources.XdsUpdateEvent

	listenerMutex    sync.RWMutex
	listenerCDSMutex sync.RWMutex
	// serviceName -> listenerName, listener
	OnRdsChangeListeners map[string]map[string]OnRdsChangeListener
	OnCdsChangeListeners map[string]map[string]OnCdsChangeListener

	// vhs,cluster,endpoint, listener from xds
	envoyVirtualHostMap     sync.Map
	envoyClusterMap         sync.Map
	envoyClusterEndpointMap sync.Map
	envoyListenerMap        sync.Map

	// host inbound for protocol export
	xdsHostInboundListenerAtomic atomic.Value

	// stop or not
	runningStatus atomic.Bool

	// secret cache
	secretCache *resources.SecretCache

	// demo secret cache
	// demoSecretCache *resources.SecretCache
}

func init() {
	//TODO enable dubbo mesh or not from env
	EnableDubboMesh = true
}

func GetPilotAgent(agentType PilotAgentType) (XdsAgent, error) {
	if pilotAgent == nil {
		pilotAgentMutex.Do(func() {
			pilotAgent, pilotAgentErr = NewPilotAgent(agentType)
		})
	}
	return pilotAgent, pilotAgentErr
}

func NewPilotAgent(agentType PilotAgentType) (XdsAgent, error) {
	// Get bootstrap info
	bootstrapInfo, err := bootstrap.GetBootStrapInfo()
	if err != nil {
		return nil, err
	}

	//TODO need to get stopChan from caller for shutdown graceful
	stopChan := make(chan struct{})

	updateChan := make(chan resources.XdsUpdateEvent, 8)

	//demoSecretCache := resources.InitSecretCacheFromFile()
	secretCache := resources.NewSecretCache()
	sdsClientChannel, err := channel.NewSdsClientChannel(stopChan, bootstrapInfo.SdsGrpcPath, bootstrapInfo.Node)
	if err != nil {
		return nil, err
	}

	// Init secret protocol
	secretProtocol, _ := protocol.NewSecretProtocol(secretCache)
	// Add secret listener
	sdsClientChannel.AddListener(secretProtocol.ProcessSecret, "secret")

	xdsClientChannel, err := channel.NewXdsClientChannel(stopChan, bootstrapInfo.XdsGrpcPath, bootstrapInfo.Node)
	if err != nil {
		return nil, err
	}
	// Init protocol handler
	ldsProtocol, _ := protocol.NewLdsProtocol(stopChan, updateChan, xdsClientChannel)
	rdsProtocol, _ := protocol.NewRdsProtocol(stopChan, updateChan, xdsClientChannel)
	cdsProtocol, _ := protocol.NewCdsProtocol(stopChan, updateChan, xdsClientChannel)
	edsProtocol, _ := protocol.NewEdsProtocol(stopChan, updateChan, xdsClientChannel)

	// Add protocol listener
	xdsClientChannel.AddListener(ldsProtocol.ProcessProtocol, "lds", channel.ListenerType)
	xdsClientChannel.AddListener(rdsProtocol.ProcessProtocol, "rds", channel.RouteType)
	xdsClientChannel.AddListener(cdsProtocol.ProcessProtocol, "cds", channel.ClusterType)
	xdsClientChannel.AddListener(edsProtocol.ProcessProtocol, "eds", channel.EndpointType)

	// Init pilot agent
	agent := &PilotAgent{
		bootstrapInfo:    bootstrapInfo,
		sdsClientChannel: sdsClientChannel,
		xdsClientChannel: xdsClientChannel,
		stopChan:         stopChan,
		updateChan:       updateChan,
		ldsProtocol:      ldsProtocol,
		rdsProtocol:      rdsProtocol,
		edsProtocol:      edsProtocol,
		cdsProtocol:      cdsProtocol,
		secretCache:      secretCache,
	}
	agent.runningStatus.Store(false)
	// Start xds/sds and wait
	if err := agent.Run(agentType); err != nil {
		return agent, err
	}
	// Add graceful shutdown call back
	extension.AddCustomShutdownCallback(agent.Stop)

	return agent, nil
}

func (p *PilotAgent) Run(agentType PilotAgentType) error {
	if runningStatus := p.runningStatus.Load(); runningStatus {
		logger.Info("pilot agent is running already")
		return nil
	}
	// Reset running status
	p.runningStatus.Store(true)
	// Get secrets
	p.sdsClientChannel.InitSds()
	// Load XdsChannel.
	p.xdsClientChannel.InitXds()

	// Start listen xds
	go p.startUpdateEventLoop()

	// Wait secret ready
	delayRead := 10 * time.Millisecond
	for {
		select {
		case <-p.stopChan:
			return nil
		case <-time.After(delayRead):
			isReady := true
			if p.secretCache.GetRoot() == nil || p.secretCache.GetWorkload() == nil {
				isReady = false
			}
			if agentType == PilotAgentTypeServerWorkload && p.GetHostInboundListener() == nil {
				isReady = false
			}
			if isReady {
				return nil
			} else {
				logger.Infof("[Pilot Agent] try to get secret or inboundListener again and delay %d milliseconds", delayRead.Milliseconds())
				delayRead = 2 * delayRead
			}

		case <-time.After(pilotAgentWaitTimeout):
			logger.Errorf("pilot agent init and wait timeout %f seconds", pilotAgentWaitTimeout.Seconds())
			return fmt.Errorf("pilot agent init and wait timeout %f seconds", pilotAgentWaitTimeout.Seconds())
		}
	}

	return nil
}

func (p *PilotAgent) startUpdateEventLoop() {
	for {
		select {
		case <-p.stopChan:
			p.Stop()
			return
		case event, ok := <-p.updateChan:
			if !ok {
				continue
			}

			switch event.Type {
			case resources.XdsEventUpdateCDS:
				if xdsClusters, ok := event.Object.([]resources.XdsCluster); ok {
					logger.Debugf("[Pilot Agent] cds event update with cds = %s", utils.ConvertJsonString(xdsClusters))
					for _, xdsCluster := range xdsClusters {
						p.envoyClusterMap.Store(xdsCluster.Name, xdsCluster)
						p.callCdsChange(xdsCluster.Name)
					}
				}

			case resources.XdsEventUpdateEDS:
				if xdsClusterEndpoints, ok := event.Object.([]resources.XdsClusterEndpoint); ok {
					logger.Debugf("[Pilot Agent] eds event update with eds = %s", utils.ConvertJsonString(xdsClusterEndpoints))
					for _, xdsClusterEndpoint := range xdsClusterEndpoints {
						p.envoyClusterEndpointMap.Store(xdsClusterEndpoint.Name, xdsClusterEndpoint)
						p.callCdsChange(xdsClusterEndpoint.Name)
					}
				}

			case resources.XdsEventUpdateLDS:
				if xdsListeners, ok := event.Object.([]resources.XdsListener); ok {
					logger.Debugf("[Pilot Agent] lds event update with lds = %s", utils.ConvertJsonString(xdsListeners))
					for _, xdsListener := range xdsListeners {
						p.envoyClusterMap.Store(xdsListener.Name, xdsListener)
						if xdsListener.IsVirtualInbound {
							// store host inbound listener
							xdsHostInboundListener := &resources.XdsHostInboundListener{
								MutualTLSMode:   xdsListener.InboundTLSMode.GetMutualTLSMode(),
								TransportSocket: xdsListener.InboundDownstreamTransportSocket,
								JwtAuthnFilter:  xdsListener.JwtAuthnFilter,
								RBACFilter:      xdsListener.RBACFilter,
							}
							logger.Infof("[Pilot Agent] update InboundListener :%s", utils.ConvertJsonString(xdsHostInboundListener))
							p.SetHostInboundListener(xdsHostInboundListener)
						}
					}
				}
			case resources.XdsEventUpdateRDS:
				if xdsRouteConfigurations, ok := event.Object.([]resources.XdsRouteConfig); ok {
					logger.Debugf("[Pilot Agent] rds event update with rds = %s", utils.ConvertJsonString(xdsRouteConfigurations))
					for _, xdsRouteConfiguration := range xdsRouteConfigurations {
						for _, xdsVirtualHost := range xdsRouteConfiguration.VirtualHosts {
							p.envoyVirtualHostMap.Store(xdsVirtualHost.Name, xdsVirtualHost)
							p.callRdsChange(xdsVirtualHost.Name, xdsVirtualHost)
						}
					}
				}
			}

		}

	}
}

//func (p *PilotAgent) GetSecretCache() *resources.SecretCache {
//	return p.secretCache
//}

func (p *PilotAgent) GetWorkloadCertificateProvider() WorkloadCertificateProvider {
	return p.secretCache
}

func (p *PilotAgent) callCdsChange(clusterName string) {
	logger.Infof("[Pilot Agent] callCdsChange clusterName:%s", clusterName)
	p.listenerMutex.RLock()
	defer p.listenerMutex.RUnlock()
	if listeners, ok := p.OnCdsChangeListeners[clusterName]; ok {
		xdsCluster, ok1 := p.envoyClusterMap.Load(clusterName)
		xdsClusterEndpoint, ok2 := p.envoyClusterEndpointMap.Load(clusterName)
		for listenerName, listener := range listeners {
			if ok1 && ok2 {
				logger.Debugf("[Pilot Agent] callCdsChange clusterName %s listener %s with cluster = %s and  eds = %s", clusterName, listenerName, utils.ConvertJsonString(xdsCluster.(resources.XdsCluster)), utils.ConvertJsonString(xdsClusterEndpoint.(resources.XdsClusterEndpoint)))
				go listener(clusterName, xdsCluster.(resources.XdsCluster), xdsClusterEndpoint.(resources.XdsClusterEndpoint))
			}
		}
	}
}

func (p *PilotAgent) callRdsChange(serviceName string, xdsVirtualHost resources.XdsVirtualHost) {
	logger.Infof("[Pilot Agent] callCdsChange serivceName:%s", serviceName)
	p.listenerMutex.RLock()
	defer p.listenerMutex.RUnlock()
	if listeners, ok := p.OnRdsChangeListeners[serviceName]; ok {
		for listenerName, listener := range listeners {
			logger.Debugf("[Pilot Agent] callRdsChange serviceName %s istener %s with rds = %s", serviceName, listenerName, utils.ConvertJsonString(xdsVirtualHost))
			go listener(serviceName, xdsVirtualHost)
		}
	}
}

func (p *PilotAgent) SubscribeRds(serviceName, listenerName string, listener OnRdsChangeListener) {
	logger.Infof("[Pilot Agent] recv SubscribeRds serviceName:%s, listenerName:%s", serviceName, listenerName)
	p.listenerMutex.Lock()
	defer p.listenerMutex.Unlock()

	if p.OnRdsChangeListeners == nil {
		p.OnRdsChangeListeners = make(map[string]map[string]OnRdsChangeListener)
	}
	if p.OnRdsChangeListeners[serviceName] == nil {
		p.OnRdsChangeListeners[serviceName] = make(map[string]OnRdsChangeListener)
	}
	p.OnRdsChangeListeners[serviceName][listenerName] = listener

	if xdsVirtualHost, ok := p.envoyVirtualHostMap.Load(serviceName); ok {
		logger.Debugf("[Pilot Agent] callRdsChange serviceName, listener %s with rds = %s", serviceName, listenerName, utils.ConvertJsonString(xdsVirtualHost.(resources.XdsVirtualHost)))
		go listener(serviceName, xdsVirtualHost.(resources.XdsVirtualHost))
	}
}

func (p *PilotAgent) UnsubscribeRds(serviceName, listenerName string) {
	p.listenerMutex.Lock()
	defer p.listenerMutex.Unlock()

	if listeners, ok := p.OnRdsChangeListeners[serviceName]; ok {
		delete(listeners, listenerName)
		if len(listeners) == 0 {
			delete(p.OnRdsChangeListeners, serviceName)
		}
	}
}

func (p *PilotAgent) SubscribeCds(clusterName, listenerName string, listener OnCdsChangeListener) {
	logger.Infof("[Pilot Agent] recv SubscribeCds clusterName:%s, listenerName:%s", clusterName, listenerName)
	p.listenerCDSMutex.Lock()
	defer p.listenerCDSMutex.Unlock()
	if p.OnCdsChangeListeners == nil {
		p.OnCdsChangeListeners = make(map[string]map[string]OnCdsChangeListener)
	}
	if p.OnCdsChangeListeners[clusterName] == nil {
		p.OnCdsChangeListeners[clusterName] = make(map[string]OnCdsChangeListener)
	}
	p.OnCdsChangeListeners[clusterName][listenerName] = listener

	xdsCluster, ok1 := p.envoyClusterMap.Load(clusterName)
	xdsClusterEndpoint, ok2 := p.envoyClusterEndpointMap.Load(clusterName)
	if ok1 && ok2 {
		logger.Debugf("[Pilot Agent] callCdsChange clusterName %s listener %s with cluster = %s and  eds = %s", clusterName, listenerName, utils.ConvertJsonString(xdsCluster.(resources.XdsCluster)), utils.ConvertJsonString(xdsClusterEndpoint.(resources.XdsClusterEndpoint)))
		go listener(clusterName, xdsCluster.(resources.XdsCluster), xdsClusterEndpoint.(resources.XdsClusterEndpoint))
	}
}

func (p *PilotAgent) UnsubscribeCds(clusterName, listenerName string) {
	p.listenerCDSMutex.Lock()
	defer p.listenerCDSMutex.Unlock()

	if listeners, ok := p.OnCdsChangeListeners[clusterName]; ok {
		delete(listeners, listenerName)
		if len(listeners) == 0 {
			delete(p.OnCdsChangeListeners, clusterName)
		}
	}
}

func (p *PilotAgent) SetHostInboundListener(xdsHostInboundListener *resources.XdsHostInboundListener) {
	p.xdsHostInboundListenerAtomic.Store(xdsHostInboundListener)
}

func (p *PilotAgent) GetHostInboundListener() *resources.XdsHostInboundListener {
	value := p.xdsHostInboundListenerAtomic.Load()
	if value != nil {
		if xdsHostInboundListener, ok := value.(*resources.XdsHostInboundListener); ok {
			return xdsHostInboundListener
		}
	}
	return nil
}

func (p *PilotAgent) GetHostInboundMutualTLSMode() resources.MutualTLSMode {
	value := p.xdsHostInboundListenerAtomic.Load()
	if value != nil {
		if xdsHostInboundListener, ok := value.(*resources.XdsHostInboundListener); ok {
			return xdsHostInboundListener.MutualTLSMode
		}
	}
	return resources.MTLSUnknown
}

func (p *PilotAgent) GetHostInboundJwtAuthentication() *resources.JwtAuthentication {
	value := p.xdsHostInboundListenerAtomic.Load()
	if value != nil {
		if xdsHostInboundListener, ok := value.(*resources.XdsHostInboundListener); ok {
			return xdsHostInboundListener.JwtAuthnFilter.JwtAuthentication
		}
	}
	return nil
}

func (p *PilotAgent) GetHostInboundRBAC() *rbacv3.RBAC {
	value := p.xdsHostInboundListenerAtomic.Load()
	if value != nil {
		if xdsHostInboundListener, ok := value.(*resources.XdsHostInboundListener); ok {
			return xdsHostInboundListener.RBACFilter.RBAC
		}
	}
	return nil
}

func (p *PilotAgent) Stop() {
	if runningStatus := p.runningStatus.Load(); runningStatus {
		// make sure stop once
		p.runningStatus.Store(false)
		logger.Infof("[Pilot Agent] Stop now...")
		close(p.stopChan)
		close(p.updateChan)
		p.sdsClientChannel.Stop()
		p.xdsClientChannel.Stop()
	}
}
