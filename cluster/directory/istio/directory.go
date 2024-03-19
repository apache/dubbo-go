package istio

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/istio"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"github.com/dubbogo/gost/log/logger"
	perrors "github.com/pkg/errors"
	"strconv"
	"strings"
	"sync"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/directory/base"
	"dubbo.apache.org/dubbo-go/v3/cluster/router/chain"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
)

type directory struct {
	*base.Directory
	invokers          []protocol.Invoker
	serviceNames      []string
	protocolName      string
	protocol          protocol.Protocol
	pilotAgent        *istio.PilotAgent
	xdsVirtualHostMap sync.Map
	xdsClusterMap     sync.Map
	serviceInterface  string
}

//super(directory.getConsumerUrl(), true);
//this.serviceType = directory.getInterface();
//this.url = directory.getConsumerUrl();
//this.applicationNames = url.getParameter("provided-by").split(",");
//this.protocolName = url.getParameter("protocol", "dubbo");
//this.protocol = directory.getProtocol();
//super.routerChain = directory.getRouterChain();

// NewDirectory Create a new staticDirectory with invokers
func NewDirectory(invokers []protocol.Invoker) *directory {
	var url *common.URL

	if len(invokers) > 0 {
		url = invokers[0].GetURL()
	}

	serviceType := url.GetParam(constant.InterfaceKey, "")
	serviceNames := strings.Split(url.GetParam(constant.ProvidedBy, ""), ",")
	//protocolName := url.GetParam(constant.ProtocolKey, "tri")
	protocolName := "tri"
	pilotAgent, err := istio.GetPilotAgent(istio.PilotAgentTypeClientWorkload)
	if err != nil {
		logger.Errorf("[xds directory] can not get pilot agent")
	}

	dir := &directory{
		Directory:        base.NewDirectory(url),
		invokers:         invokers,
		serviceInterface: serviceType,
		serviceNames:     serviceNames,
		protocolName:     protocolName,
		pilotAgent:       pilotAgent,
	}
	for _, serviceName := range serviceNames {
		pilotAgent.SubscribeRds(serviceName, "rdsDirectory", dir.OnRdsChangeListener)
	}
	dir.RouterChain().SetInvokers(invokers)
	return dir
}

func (dir *directory) OnRdsChangeListener(serviceName string, xdsVirtualHost resources.XdsVirtualHost) error {
	// Get old clusters
	oldClusters := dir.getAllClusters()
	// Update xdsVirtualHostMap
	dir.xdsVirtualHostMap.Store(serviceName, xdsVirtualHost)
	// Get new clusters
	newClusters := dir.getAllClusters()

	// Perform necessary actions based on cluster changes
	dir.changeClusterSubscribe(oldClusters, newClusters)

	return nil
}

func (dir *directory) changeClusterSubscribe(oldCluster, newCluster []string) {
	// Create sets of old and new clusters
	oldSet := make(map[string]bool)
	newSet := make(map[string]bool)
	for _, cluster := range oldCluster {
		oldSet[cluster] = true
	}
	for _, cluster := range newCluster {
		newSet[cluster] = true
	}

	// Find clusters to remove subscription
	removeSubscribe := make([]string, 0)
	for _, cluster := range oldCluster {
		if !newSet[cluster] {
			removeSubscribe = append(removeSubscribe, cluster)
		}
	}
	// Remove subscription for clusters that are no longer used
	for _, cluster := range removeSubscribe {
		dir.pilotAgent.UnsubscribeCds(cluster, "rdsDirectory")
		dir.xdsClusterMap.Delete(cluster)
		// todo remove inokers which is belong to unsubscribed cluster
	}

	// Find clusters to add subscription
	addSubscribe := make([]string, 0)
	for _, cluster := range newCluster {
		if !oldSet[cluster] {
			addSubscribe = append(addSubscribe, cluster)
		}
	}

	// Add subscription for new clusters
	for _, cluster := range addSubscribe {
		dir.pilotAgent.SubscribeCds(cluster, "rdsDirectory", dir.OnEdsChangeListener)
	}
}

func (dir *directory) OnEdsChangeListener(clusterName string, xdsCluster resources.XdsCluster, xdsClusterEndpoint resources.XdsClusterEndpoint) error {

	// Get lbPolicy and xdsEndpoints
	//lbPolicy := xdsCluster.LbPolicy
	mutualTLSMode := xdsCluster.TlsMode.GetMutualTLSMode()
	xdsEndpoints := xdsClusterEndpoint.Endpoints

	// Create a list to hold invokers
	invokers := make([]protocol.Invoker, 0)
	baseUrl := dir.GetURL()

	// Iterate through xdsEndpoints
	for _, e := range xdsEndpoints {
		ip := e.Address
		port := e.Port
		// Construct URL for invoker
		url := common.NewURLWithOptions(
			common.WithProtocol(dir.protocolName),
			common.WithIp(ip),
			common.WithPort(strconv.Itoa(int(port))),
			common.WithPath(baseUrl.Path),
			common.WithParamsValue(constant.InterfaceKey, dir.serviceInterface),
			// load balance here
			//common.WithParamsValue(constant.LoadbalanceKey, ""),
			// xds and MutualTLSMode
			common.WithParamsValue(constant.XdsKey, "true"),
			common.WithParamsValue(constant.MutualTLSModeKey, resources.MutualTLSModeToString(mutualTLSMode)),
			common.WithParamsValue(constant.ClusterIDKey, xdsCluster.Name),
			// client spiffe match
			common.WithParamsValue(constant.TLSSubjectAltNamesMatchKey, xdsCluster.TransportSocket.SubjectAltNamesMatch),
			common.WithParamsValue(constant.TLSSubjectAltNamesValueKey, xdsCluster.TransportSocket.SubjectAltNamesValue),
		)
		// Refer to the protocol to create an invoker
		invoker := dir.protocol.Refer(url)
		invokers = append(invokers, invoker)
	}
	// Add invokers to directory's invoker list
	dir.invokers = append(dir.invokers, invokers...)

	// Set invokers for xdsCluster
	xdsCluster.Invokers = invokers
	// Update xdsClusterMap
	dir.xdsClusterMap.Store(clusterName, xdsCluster)
	return nil
}

func (dir *directory) getAllClusters() []string {
	clusters := make([]string, 0)
	clustersMap := make(map[string]bool)
	dir.xdsVirtualHostMap.Range(func(key, value interface{}) bool {
		virtualHost := value.(resources.XdsVirtualHost)
		for _, route := range virtualHost.Routes {
			action := route.Action
			if len(action.Cluster) > 0 {
				clusterName := action.Cluster
				if _, ok := clustersMap[clusterName]; !ok {
					clustersMap[clusterName] = true
					clusters = append(clusters, clusterName)
				}
			} else {
				for _, clusterWeight := range action.ClusterWeights {
					clusterName := clusterWeight.Name
					if _, ok := clustersMap[clusterName]; !ok {
						clustersMap[clusterName] = true
						clusters = append(clusters, clusterName)
					}
				}
			}

		}
		return true
	})

	return clusters
}

// for-loop invokers ,if all invokers is available ,then it means directory is available
func (dir *directory) IsAvailable() bool {
	if dir.Directory.IsDestroyed() {
		return false
	}

	if len(dir.invokers) == 0 {
		return false
	}
	for _, invoker := range dir.invokers {
		if !invoker.IsAvailable() {
			return false
		}
	}
	return true
}

// List List invokers
func (dir *directory) List(invocation protocol.Invocation) []protocol.Invoker {
	l := len(dir.invokers)
	invokers := make([]protocol.Invoker, l)
	copy(invokers, dir.invokers)
	routerChain := dir.RouterChain()

	if routerChain == nil {
		return invokers
	}
	dirUrl := dir.GetURL()
	return routerChain.Route(dirUrl, invocation)
}

// Destroy Destroy
func (dir *directory) Destroy() {
	dir.Directory.DoDestroy(func() {
		for _, ivk := range dir.invokers {
			ivk.Destroy()
		}
		dir.invokers = []protocol.Invoker{}
	})
}

// BuildRouterChain build router chain by invokers
func (dir *directory) BuildRouterChain(invokers []protocol.Invoker) error {
	if len(invokers) == 0 {
		return perrors.Errorf("invokers == null")
	}
	routerChain, e := chain.NewRouterChain()
	if e != nil {
		return e
	}
	routerChain.SetInvokers(dir.invokers)
	dir.SetRouterChain(routerChain)
	return nil
}

func (dir *directory) Subscribe(url *common.URL) error {
	panic("Static directory does not support subscribing to registry.")
}
