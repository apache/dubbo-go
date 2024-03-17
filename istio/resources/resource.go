package resources

type XdsEventUpdateType uint32

const (
	XdsEventUpdateLDS XdsEventUpdateType = iota
	XdsEventUpdateRDS
	XdsEventUpdateCDS
	XdsEventUpdateEDS
)

type XdsUpdateEvent struct {
	Type   XdsEventUpdateType
	Object interface{}
}

type XdsEndpoint struct {
	ClusterName string
	Protocol    string
	Address     string
	Port        uint32
	Healthy     bool
	Weight      int
}

type XdsClusterWeight struct {
	Name   string
	Weight uint32
}

type XdsRoute struct {
	Name   string
	Match  XdsRouteMatch
	Action XdsRouteAction
}

type XdsRouteMatch struct {
	Path          string
	Prefix        string
	Regex         string
	CaseSensitive bool
}

type XdsRouteAction struct {
	Cluster        string
	ClusterWeights []XdsClusterWeight
}

type XdsVirtualHost struct {
	Name    string
	Domains []string
	Routes  []XdsRoute
}

type XdsRouteConfig struct {
	Name         string
	VirtualHosts map[string]XdsVirtualHost
}

type XdsListener struct {
	Name             string
	HasRds           bool
	RdsResourceNames []string
	TrafficDirection string
	// virtual inbound 15006 listener
	IsVirtualInbound bool
	//FilterChains     []XdsFilterChain
	// virtual inbound 15006 listener tls and downstream transport socket which is for mtls
	InboundTLSMode                   XdsTLSMode
	InboundDownstreamTransportSocket XdsDownstreamTransportSocket
}

type XdsDownstreamTransportSocket struct {
	SubjectAltNamesMatch string // exact, prefix
	SubjectAltNamesValue string
	//tlsContext               sockets_tls_v3.DownstreamTlsContext
	RequireClientCertificate bool
}

type XdsFilterChain struct {
	Name             string
	FilterChainMatch XdsFilterChainMatch
	TransportSocket  XdsDownstreamTransportSocket
	Filters          []XdsFilter
}

type XdsFilterChainMatch struct {
	DestinationPort   uint32
	TransportProtocol string
}

type XdsFilter struct {
	Name                         string
	IncludeHttpConnectionManager bool
	HasRds                       bool
	RdsResourceName              string
}

type XdsHostInboundListener struct {
	MutualTLSMode   MutualTLSMode
	TransportSocket XdsDownstreamTransportSocket
	// other host inbound info for protocol export here
}
