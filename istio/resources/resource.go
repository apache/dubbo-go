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

type EnvoyEndpoint struct {
	ClusterName string
	Protocol    string
	Address     string
	Port        uint32
	Healthy     bool
	Weight      int
}

type EnvoyClusterWeight struct {
	Name   string
	Weight uint32
}

type EnvoyRoute struct {
	Name   string
	Match  EnvoyRouteMatch
	Action EnvoyRouteAction
}

type EnvoyRouteMatch struct {
	Path          string
	Prefix        string
	Regex         string
	CaseSensitive bool
}

type EnvoyRouteAction struct {
	Cluster        string
	ClusterWeights []EnvoyClusterWeight
}

type EnvoyVirtualHost struct {
	Name    string
	Domains []string
	Routes  []EnvoyRoute
}

type EnvoyRouteConfig struct {
	Name         string
	VirtualHosts map[string]EnvoyVirtualHost
}

type EnvoyListener struct {
	Name                  string
	IsRds                 bool
	RdsResourceName       string
	IsVirtualInbound      bool
	IsVirtualInbound15006 bool
}
