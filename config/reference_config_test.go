package config

import (
	"github.com/dubbo/go-for-apache-dubbo/cluster/cluster_impl"
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/common/extension"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

var regProtocol protocol.Protocol

func doInit() {
	consumerConfig = &ConsumerConfig{
		ApplicationConfig: ApplicationConfig{
			Organization: "dubbo_org",
			Name:         "dubbo",
			Module:       "module",
			Version:      "2.6.0",
			Owner:        "dubbo",
			Environment:  "test"},
		Registries: []RegistryConfig{
			{
				Id:         "shanghai_reg1",
				Type:       "mock",
				TimeoutStr: "2s",
				Group:      "shanghai_idc",
				Address:    "127.0.0.1:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			{
				Id:         "shanghai_reg2",
				Type:       "mock",
				TimeoutStr: "2s",
				Group:      "shanghai_idc",
				Address:    "127.0.0.2:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			{
				Id:         "hangzhou_reg1",
				Type:       "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.3:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
			{
				Id:         "hangzhou_reg2",
				Type:       "mock",
				TimeoutStr: "2s",
				Group:      "hangzhou_idc",
				Address:    "127.0.0.4:2181",
				Username:   "user1",
				Password:   "pwd1",
			},
		},
		References: []ReferenceConfig{
			{
				InterfaceName: "testInterface",
				Protocol:      "mock",
				Registries:    []ConfigRegistry{"shanghai_reg1", "shanghai_reg2", "hangzhou_reg1", "hangzhou_reg2"},
				Cluster:       "failover",
				Loadbalance:   "random",
				Retries:       3,
				Group:         "huadong_idc",
				Version:       "1.0.0",
				Methods: []struct {
					Name        string `yaml:"name"  json:"name,omitempty"`
					Retries     int64  `yaml:"retries"  json:"retries,omitempty"`
					Loadbalance string `yaml:"loadbalance"  json:"loadbalance,omitempty"`
				}{
					{
						Name:        "GetUser",
						Retries:     2,
						Loadbalance: "random",
					},
					{
						Name:        "GetUser1",
						Retries:     2,
						Loadbalance: "random",
					},
				},
			},
		},
	}
}

func Test_ReferMultireg(t *testing.T) {
	doInit()
	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster("registryAware", cluster_impl.NewRegistryAwareCluster)

	for _, reference := range consumerConfig.References {
		reference.Refer()
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func Test_Refer(t *testing.T) {
	doInit()
	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster("registryAware", cluster_impl.NewRegistryAwareCluster)
	consumerConfig.References[0].Registries = []ConfigRegistry{"shanghai_reg1"}

	for _, reference := range consumerConfig.References {
		reference.Refer()
		assert.NotNil(t, reference.invoker)
		assert.NotNil(t, reference.pxy)
	}
	consumerConfig = nil
}

func Test_Implement(t *testing.T) {
	doInit()
	extension.SetProtocol("registry", GetProtocol)
	extension.SetCluster("registryAware", cluster_impl.NewRegistryAwareCluster)
	for _, reference := range consumerConfig.References {
		reference.Refer()
		reference.Implement(&MockService{})
		assert.NotNil(t, reference.GetRPCService())

	}
	consumerConfig = nil
}
func GetProtocol() protocol.Protocol {
	if regProtocol != nil {
		return regProtocol
	}
	return newRegistryProtocol()
}

func newRegistryProtocol() protocol.Protocol {
	return &mockRegistryProtocol{}
}

type mockRegistryProtocol struct {
}

func (*mockRegistryProtocol) Refer(url common.URL) protocol.Invoker {
	return protocol.NewBaseInvoker(url)
}

func (*mockRegistryProtocol) Export(invoker protocol.Invoker) protocol.Exporter {
	return protocol.NewBaseExporter("test", invoker, &sync.Map{})
}

func (*mockRegistryProtocol) Destroy() {

}
