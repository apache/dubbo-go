package zookeeper

import (
	"context"
	"sync"
	"testing"
)

import (
	gxset "github.com/dubbogo/gost/container/set"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/registry"
	"dubbo.apache.org/dubbo-go/v3/registry/event"
)

func Test_newZookeeperServiceDiscovery(t *testing.T) {
	url, _ := common.NewURL("dubbo://127.0.0.1:2181",
		common.WithParamsValue(constant.ClientNameKey, "zk-client"))
	sd, err := newZookeeperServiceDiscovery(url)
	assert.Nil(t, err)
	err = sd.Destroy()
	assert.Nil(t, err)

}
func TestFunction(t *testing.T) {

	extension.SetProtocol("mock", func() protocol.Protocol {
		return &mockProtocol{}
	})

	url, _ := common.NewURL("dubbo://127.0.0.1:2181")
	sd, _ := newZookeeperServiceDiscovery(url)
	defer func() {
		_ = sd.Destroy()
	}()

	ins := &registry.DefaultServiceInstance{
		ID:       "testID",
		Host:     "127.0.0.1",
		Port:     2181,
		Enable:   true,
		Healthy:  true,
		Metadata: nil,
	}
	ins.Metadata = map[string]string{"t1": "test12", constant.MetadataServiceURLParamsPropertyName: `{"protocol":"mock","timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"2181"}`}
	err := sd.Register(ins)
	assert.Nil(t, err)

	wg := &sync.WaitGroup{}
	wg.Add(1)
	hs := gxset.NewSet()

	sicl := event.NewServiceInstancesChangedListener(hs)

	err = sd.AddListener(sicl)
	assert.NoError(t, err)

	ins = &registry.DefaultServiceInstance{
		ID:       "testID",
		Host:     "127.0.0.1",
		Port:     2181,
		Enable:   true,
		Healthy:  true,
		Metadata: nil,
	}
	ins.Metadata = map[string]string{"t1": "test12", constant.MetadataServiceURLParamsPropertyName: `{"protocol":"mock","timeout":"10000","version":"1.0.0","dubbo":"2.0.2","release":"2.7.6","port":"2181"}`}
	err = sd.Update(ins)
	assert.NoError(t, err)
	err = sd.Unregister(ins)
	assert.Nil(t, err)
}

type testNotify struct {
	wg *sync.WaitGroup
	t  *testing.T
}

func (tn *testNotify) Notify(e *registry.ServiceEvent) {
	assert.Equal(tn.t, "2181", e.Service.Port)
	tn.wg.Done()
}

func (tn *testNotify) NotifyAll([]*registry.ServiceEvent, func()) {}

type mockProtocol struct{}

func (m mockProtocol) Export(protocol.Invoker) protocol.Exporter {
	panic("implement me")
}

func (m mockProtocol) Refer(*common.URL) protocol.Invoker {
	return &mockInvoker{}
}

func (m mockProtocol) Destroy() {
	panic("implement me")
}

type mockInvoker struct{}

func (m *mockInvoker) GetURL() *common.URL {
	panic("implement me")
}

func (m *mockInvoker) IsAvailable() bool {
	panic("implement me")
}

func (m *mockInvoker) Destroy() {
	panic("implement me")
}

func (m *mockInvoker) Invoke(context.Context, protocol.Invocation) protocol.Result {
	// for getMetadataInfo and ServiceInstancesChangedListenerImpl onEvent
	serviceInfo := &common.ServiceInfo{ServiceKey: "test", MatchKey: "test"}
	services := make(map[string]*common.ServiceInfo)
	services["test"] = serviceInfo
	return &protocol.RPCResult{
		Rest: &common.MetadataInfo{
			Services: services,
		},
	}
}
