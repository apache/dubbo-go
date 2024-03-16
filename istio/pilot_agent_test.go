package istio

import (
	"dubbo.apache.org/dubbo-go/v3/istio/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	"testing"
)

func TestGetPilotAgent(t *testing.T) {
	bootstrapInfo, _ := bootstrap.GetBootStrapInfo()
	fmt.Sprintf("bootstrapinfo %v", bootstrapInfo)
	bootstrapInfo.SdsGrpcPath = "/Users/jun/GolandProjects/dubbo/dubbo-mesh/var/run/dubbomesh/workload-spiffe-uds/socket"
	bootstrapInfo.XdsGrpcPath = "/Users/jun/GolandProjects/dubbo/dubbo-mesh/var/run/dubbomesh/proxy/XDS"
	pilotAgent, _ := GetPilotAgent()
	OnRdsChangeListener := func(serviceName string, xdsVirtualHost resources.EnvoyVirtualHost) error {
		logger.Infof("OnRdsChangeListener serviceName %s with rds = %s", serviceName, utils.ConvertJsonString(xdsVirtualHost))
		return nil
	}
	OnEdsChangeListener := func(clusterName string, xdsCluster resources.EnvoyCluster, xdsClusterEndpoint resources.EnvoyClusterEndpoint) error {
		logger.Infof("OnEdsChangeListener clusterName %s with cluster = %s and  eds = %s", clusterName, utils.ConvertJsonString(xdsCluster), utils.ConvertJsonString(xdsClusterEndpoint))
		return nil
	}
	pilotAgent.SubscribeCds("outbound|8000||httpbin.foo.svc.cluster.local", "eds", OnEdsChangeListener)
	pilotAgent.SubscribeRds("8000", "rds", OnRdsChangeListener)
	select {}

}
