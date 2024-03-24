package resources

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type XdsCluster struct {
	Type            string
	Name            string
	LbPolicy        string
	Invokers        []protocol.Invoker
	Service         XdsClusterService
	TransportSocket XdsUpstreamTransportSocket
	TlsMode         XdsTLSMode
}

type XdsClusterEndpoint struct {
	Name      string
	Endpoints []XdsEndpoint
}

type XdsEndpoint struct {
	ClusterName string
	Protocol    string
	Address     string
	Port        uint32
	Healthy     bool
	Weight      int
}

// filter from metadata
type XdsClusterService struct {
	Name      string
	Namespace string
	Host      string
}

type XdsUpstreamTransportSocket struct {
	SubjectAltNamesMatch string // exact, prefix,  contains
	SubjectAltNamesValue string
	//tlsContext           sockets_tls_v3.UpstreamTlsContext
}
