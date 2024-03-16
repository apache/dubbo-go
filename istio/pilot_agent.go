package istio

import (
	"dubbo.apache.org/dubbo-go/v3/istio/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/istio/channel"
	"dubbo.apache.org/dubbo-go/v3/istio/protocol"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	"sync"
	"time"
)

var (
	EnableDubboMesh bool
	pilotAgent      *PilotAgent
	pilotAgentMutex sync.Once
	pilotAgentErr   error
)

const (
	pilotAgentWaitTimeout = 10 * time.Second
)

type OnRdsChangeListener func(serviceName string, xdsVirtualHost resources.XdsVirtualHost) error
type OnEdsChangeListener func(clusterName string, xdsCluster resources.XdsCluster, xdsClusterEndpoint resources.XdsClusterEndpoint) error

type PilotAgent struct {
	bootstrapInfo    *bootstrap.BootstrapInfo
	sdsClientChannel *channel.SdsClientChannel
	xdsClientChannel *channel.XdsClientChannel
	secretCache      resources.SecretCache
	cdsProtocol      *protocol.CdsProtocol
	edsProtocol      *protocol.EdsProtocol
	ldsProtocol      *protocol.LdsProtocol
	rdsProtocol      *protocol.RdsProtocol
	stopChan         chan struct{}
	updateChan       chan resources.XdsUpdateEvent

	listenerMutex sync.RWMutex
	// serviceName -> listenerName, listener
	OnRdsChangeListeners map[string]map[string]OnRdsChangeListener
	OnCdsChangeListeners map[string]map[string]OnEdsChangeListener

	envoyVirtualHostMap     sync.Map
	envoyClusterMap         sync.Map
	envoyClusterEndpointMap sync.Map
	envoyListenerMap        sync.Map
}

func init() {
	//TODO enable dubbo mesh or not from env
	EnableDubboMesh = true
}

func GetPilotAgent() (*PilotAgent, error) {
	if pilotAgent == nil && EnableDubboMesh {
		pilotAgentMutex.Do(func() {
			pilotAgent, pilotAgentErr = NewPilotAgent()
		})
	}
	return pilotAgent, pilotAgentErr
}

func NewPilotAgent() (*PilotAgent, error) {
	// Get bootstrap info
	bootstrapInfo, err := bootstrap.GetBootStrapInfo()
	if err != nil {
		return nil, err
	}

	stopChan := make(chan struct{})

	updateChan := make(chan resources.XdsUpdateEvent, 8)

	sdsClientChannel, err := channel.NewSdsClientChannel(stopChan, bootstrapInfo.SdsGrpcPath, bootstrapInfo.Node)
	if err != nil {
		return nil, err
	}
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
	pilotAgent := &PilotAgent{
		bootstrapInfo:    bootstrapInfo,
		sdsClientChannel: sdsClientChannel,
		xdsClientChannel: xdsClientChannel,
		stopChan:         stopChan,
		updateChan:       updateChan,
		ldsProtocol:      ldsProtocol,
		rdsProtocol:      rdsProtocol,
		edsProtocol:      edsProtocol,
		cdsProtocol:      cdsProtocol,
	}
	// Start xds/sds and wait
	go pilotAgent.initAndWait()
	return pilotAgent, nil
}

func (p *PilotAgent) initAndWait() error {
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
		case <-time.After(delayRead):
			if p.secretCache.GetRoot() != nil && p.secretCache.GetWorkload() != nil {
				return nil
			} else {
				logger.Infof("try to get pilot agent secret again and delay %d milliseconds", delayRead.Milliseconds())
				delayRead = 2 * delayRead
			}

		case <-time.After(pilotAgentWaitTimeout):
			return fmt.Errorf("pilot agent init and wait timeout %f seconds", pilotAgentWaitTimeout.Seconds())
		}
	}

	<-p.stopChan

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
					logger.Infof("pilot agent cds event update with cds = %s", utils.ConvertJsonString(xdsClusters))
					for _, xdsCluster := range xdsClusters {
						p.envoyClusterMap.Store(xdsCluster.Name, xdsCluster)
						p.callEdsChange(xdsCluster.Name)
					}
				}

			case resources.XdsEventUpdateEDS:
				if xdsClusterEndpoints, ok := event.Object.([]resources.XdsClusterEndpoint); ok {
					logger.Infof("pilot agent eds event update with eds = %s", utils.ConvertJsonString(xdsClusterEndpoints))
					for _, xdsClusterEndpoint := range xdsClusterEndpoints {
						p.envoyClusterEndpointMap.Store(xdsClusterEndpoint.Name, xdsClusterEndpoint)
						p.callEdsChange(xdsClusterEndpoint.Name)
					}
				}

			case resources.XdsEventUpdateLDS:
				if xdsListeners, ok := event.Object.([]resources.XdsListener); ok {
					logger.Infof("pilot agent lds event update with lds = %s", utils.ConvertJsonString(xdsListeners))
					for _, xdsListener := range xdsListeners {
						p.envoyClusterMap.Store(xdsListener.Name, xdsListener)
					}
				}
			case resources.XdsEventUpdateRDS:
				if xdsRouteConfigurations, ok := event.Object.([]resources.XdsRouteConfig); ok {
					logger.Infof("pilot agent rds event update with rds = %s", utils.ConvertJsonString(xdsRouteConfigurations))
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

func (p *PilotAgent) callEdsChange(clusterName string) {
	p.listenerMutex.RLock()
	defer p.listenerMutex.RUnlock()
	xdsCluster, ok1 := p.envoyClusterMap.Load(clusterName)
	xdsClusterEndpoint, ok2 := p.envoyClusterEndpointMap.Load(clusterName)

	if listeners, ok := p.OnCdsChangeListeners[clusterName]; ok {
		for listenerName, listener := range listeners {
			if ok1 && ok2 {
				logger.Infof("pilot agent callEdsChange clusterName %s listener %s with cluster = %s and  eds = %s", clusterName, listenerName, utils.ConvertJsonString(xdsCluster.(resources.XdsCluster)), utils.ConvertJsonString(xdsClusterEndpoint.(resources.XdsClusterEndpoint)))
				listener(clusterName, xdsCluster.(resources.XdsCluster), xdsClusterEndpoint.(resources.XdsClusterEndpoint))
			}
		}
	}
}

func (p *PilotAgent) callRdsChange(serviceName string, xdsVirtualHost resources.XdsVirtualHost) {
	p.listenerMutex.RLock()
	defer p.listenerMutex.RUnlock()
	if listeners, ok := p.OnRdsChangeListeners[serviceName]; ok {
		for listenerName, listener := range listeners {
			logger.Infof("pilot agent callRdsChange serviceName %s istener %s with rds = %s", serviceName, listenerName, utils.ConvertJsonString(xdsVirtualHost))
			listener(serviceName, xdsVirtualHost)
		}
	}
}

func (p *PilotAgent) SubscribeRds(serviceName, listenerName string, listener OnRdsChangeListener) {
	func() {
		p.listenerMutex.Lock()
		defer p.listenerMutex.Unlock()

		if p.OnRdsChangeListeners == nil {
			p.OnRdsChangeListeners = make(map[string]map[string]OnRdsChangeListener)
		}
		if p.OnRdsChangeListeners[serviceName] == nil {
			p.OnRdsChangeListeners[serviceName] = make(map[string]OnRdsChangeListener)
		}
		p.OnRdsChangeListeners[serviceName][listenerName] = listener
	}()

	if xdsVirtualHost, ok := p.envoyVirtualHostMap.Load(serviceName); ok {
		logger.Infof("pilot agent callRdsChange serviceName, listener %s with rds = %s", serviceName, listenerName, utils.ConvertJsonString(xdsVirtualHost.(resources.XdsVirtualHost)))
		listener(serviceName, xdsVirtualHost.(resources.XdsVirtualHost))
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

func (p *PilotAgent) SubscribeCds(clusterName, listenerName string, listener OnEdsChangeListener) {
	func() {
		p.listenerMutex.Lock()
		defer p.listenerMutex.Unlock()

		if p.OnCdsChangeListeners == nil {
			p.OnCdsChangeListeners = make(map[string]map[string]OnEdsChangeListener)
		}
		if p.OnCdsChangeListeners[clusterName] == nil {
			p.OnCdsChangeListeners[clusterName] = make(map[string]OnEdsChangeListener)
		}
		p.OnCdsChangeListeners[clusterName][listenerName] = listener
	}()

	xdsCluster, ok1 := p.envoyClusterMap.Load(clusterName)
	xdsClusterEndpoint, ok2 := p.envoyClusterEndpointMap.Load(clusterName)
	if ok1 && ok2 {
		logger.Infof("pilot agent callEdsChange clusterName %s listener %s with cluster = %s and  eds = %s", clusterName, listenerName, utils.ConvertJsonString(xdsCluster.(resources.XdsCluster)), utils.ConvertJsonString(xdsClusterEndpoint.(resources.XdsClusterEndpoint)))
		listener(clusterName, xdsCluster.(resources.XdsCluster), xdsClusterEndpoint.(resources.XdsClusterEndpoint))
	}
}

func (p *PilotAgent) UnsubscribeCds(clusterName, listenerName string) {
	p.listenerMutex.Lock()
	defer p.listenerMutex.Unlock()

	if listeners, ok := p.OnCdsChangeListeners[clusterName]; ok {
		delete(listeners, listenerName)
		if len(listeners) == 0 {
			delete(p.OnCdsChangeListeners, clusterName)
		}
	}
}

func (p *PilotAgent) Stop() {
	close(p.updateChan)
}
