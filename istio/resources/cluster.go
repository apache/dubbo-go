package resources

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

// MutualTLSMode is the mutual TLS mode specified by authentication policy.
type MutualTLSMode int

const (
	// MTLSUnknown is used to indicate the variable hasn't been initialized correctly (with the authentication policy).
	MTLSUnknown MutualTLSMode = iota

	// MTLSDisable if authentication policy disable mTLS.
	MTLSDisable

	// MTLSPermissive if authentication policy enable mTLS in permissive mode.
	MTLSPermissive

	// MTLSStrict if authentication policy enable mTLS in strict mode.
	MTLSStrict
)

type XdsTlsMode struct {
	IsTls       bool
	IsRawBuffer bool
}

func (t XdsTlsMode) GetMutualTLSMode() MutualTLSMode {
	if t.IsTls && t.IsRawBuffer {
		return MTLSPermissive
	}

	if t.IsTls {
		return MTLSStrict
	}

	return MTLSDisable
}

type XdsCluster struct {
	Type            string
	Name            string
	LbPolicy        string
	Invokers        []protocol.Invoker
	Service         XdsClusterService
	TransportSocket XdsUpstreamTransportSocket
	TlsMode         XdsTlsMode
}

type XdsClusterEndpoint struct {
	Name      string
	Endpoints []XdsEndpoint
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
