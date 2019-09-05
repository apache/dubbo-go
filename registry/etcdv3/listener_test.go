package etcdv3

import (
	"context"
	"github.com/apache/dubbo-go/config_center"
	"testing"
	"time"
)

import (
	"github.com/dubbogo/getty"
	"github.com/stretchr/testify/suite"
	"go.etcd.io/etcd/embed"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/remoting"
)

type RegistryTestSuite struct {
	suite.Suite
	etcd *embed.Etcd
}

// start etcd server
func (suite *RegistryTestSuite) SetupSuite() {

	t := suite.T()

	cfg := embed.NewConfig()
	cfg.Dir = "/tmp/default.etcd"
	e, err := embed.StartEtcd(cfg)
	if err != nil {
		t.Fatal(err)
	}
	select {
	case <-e.Server.ReadyNotify():
		t.Log("Server is ready!")
	case <-getty.GetTimeWheel().After(60 * time.Second):
		e.Server.Stop() // trigger a shutdown
		t.Logf("Server took too long to start!")
	}

	suite.etcd = e
	return
}

// stop etcd server
func (suite *RegistryTestSuite) TearDownSuite() {
	suite.etcd.Close()
}

func (suite *RegistryTestSuite) TestDataChange() {

	t := suite.T()

	listener := NewRegistryDataListener(&MockDataListener{})
	url, _ := common.NewURL(context.Background(), "jsonrpc%3A%2F%2F127.0.0.1%3A20001%2Fcom.ikurento.user.UserProvider%3Fanyhost%3Dtrue%26app.version%3D0.0.1%26application%3DBDTService%26category%3Dproviders%26cluster%3Dfailover%26dubbo%3Ddubbo-provider-golang-2.6.0%26environment%3Ddev%26group%3D%26interface%3Dcom.ikurento.user.UserProvider%26ip%3D10.32.20.124%26loadbalance%3Drandom%26methods.GetUser.loadbalance%3Drandom%26methods.GetUser.retries%3D1%26methods.GetUser.weight%3D0%26module%3Ddubbogo%2Buser-info%2Bserver%26name%3DBDTService%26organization%3Dikurento.com%26owner%3DZX%26pid%3D74500%26retries%3D0%26service.filter%3Decho%26side%3Dprovider%26timestamp%3D1560155407%26version%3D%26warmup%3D100")
	listener.AddInterestedURL(&url)
	if !listener.DataChange(remoting.Event{Path: "/dubbo/com.ikurento.user.UserProvider/providers/jsonrpc%3A%2F%2F127.0.0.1%3A20001%2Fcom.ikurento.user.UserProvider%3Fanyhost%3Dtrue%26app.version%3D0.0.1%26application%3DBDTService%26category%3Dproviders%26cluster%3Dfailover%26dubbo%3Ddubbo-provider-golang-2.6.0%26environment%3Ddev%26group%3D%26interface%3Dcom.ikurento.user.UserProvider%26ip%3D10.32.20.124%26loadbalance%3Drandom%26methods.GetUser.loadbalance%3Drandom%26methods.GetUser.retries%3D1%26methods.GetUser.weight%3D0%26module%3Ddubbogo%2Buser-info%2Bserver%26name%3DBDTService%26organization%3Dikurento.com%26owner%3DZX%26pid%3D74500%26retries%3D0%26service.filter%3Decho%26side%3Dprovider%26timestamp%3D1560155407%26version%3D%26warmup%3D100"}) {
		t.Fatal("data change not ok")
	}
}

func TestRegistrySuite(t *testing.T) {
	suite.Run(t, &RegistryTestSuite{})
}

type MockDataListener struct{}

func (*MockDataListener) Process(configType *config_center.ConfigChangeEvent) {}
