package k8s_api

import (
	"github.com/apache/dubbo-go/config_center"
	"github.com/apache/dubbo-go/remoting/k8sCRD"
)

const GroupName = "service.dubbo.apache.org"
const GroupVersion = "v1alpha1"
const Namespace = "dubbo-workplace"

func SetK8sEventListener(listener config_center.ConfigurationListener) error {
	vsUniformRouterListenerHandler := newVirtualServiceListenerHandler(listener)
	drUniformRouterListenerHandler := newDestRuleListenerHandler(listener)
	k8sCRDClient, err := k8sCRD.NewK8sCRDClient(GroupName, GroupVersion, Namespace, vsUniformRouterListenerHandler, drUniformRouterListenerHandler)
	if err != nil {
		return err
	}
	k8sCRDClient.WatchResources()
	return nil
}
