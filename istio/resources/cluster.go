package resources

import (
	"dubbo.apache.org/dubbo-go/v3/protocol"
	sockets_tls_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
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

type EnvoyCluster struct {
	Type     string
	Name     string
	LbPolicy string
	//Endpoints []EnvoyEndpoint
	Invokers []protocol.Invoker
	Service  EnvoyClusterService
	TlsMode  EnvoyClusterTlsMode
}

type EnvoyClusterEndpoint struct {
	Name      string
	Endpoints []EnvoyEndpoint
}

// filter from metadata
type EnvoyClusterService struct {
	Name      string
	Namespace string
	Host      string
}

type EnvoyClusterTlsMode struct {
	SubjectAltNamesMatch string // exact, prefix
	SubjectAltNamesValue string
	IsTls                bool
	IsRawBuffer          bool
	tlsContext           sockets_tls_v3.UpstreamTlsContext
}

func (t EnvoyClusterTlsMode) GetMutualTLSMode() MutualTLSMode {
	if t.IsTls && t.IsRawBuffer {
		return MTLSPermissive
	}

	if t.IsTls {
		return MTLSStrict
	}

	return MTLSDisable
}
