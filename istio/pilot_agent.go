package istio

import "dubbo.apache.org/dubbo-go/v3/istio/resources"

type PilotAgentType int32

const (
	PilotAgentTypeServerWorkload PilotAgentType = iota
	PilotAgentTypeClientWorkload
)

type OnRdsChangeListener func(serviceName string, xdsVirtualHost resources.XdsVirtualHost) error
type OnEdsChangeListener func(clusterName string, xdsCluster resources.XdsCluster, xdsClusterEndpoint resources.XdsClusterEndpoint) error

type XdsAgent interface {
	Run(pilotAgentType PilotAgentType) error
	GetSecretCache() *resources.SecretCache
	SubscribeRds(serviceName, listenerName string, listener OnRdsChangeListener)
	UnsubscribeRds(serviceName, listenerName string)
	SubscribeCds(clusterName, listenerName string, listener OnEdsChangeListener)
	UnsubscribeCds(clusterName, listenerName string)
	GetHostInboundListener() *resources.XdsHostInboundListener
	GetHostInboundMutualTLSMode() resources.MutualTLSMode
	Stop()
}
