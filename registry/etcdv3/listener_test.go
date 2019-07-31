package etcdv3

import (
	"context"
	"os"
	"os/exec"
	"testing"
)
import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/remoting"
)


func startETCDServer(t *testing.T){

	cmd := exec.Command("./load.sh",  "start")
	//cmd := exec.Command("pwd")
	cmd.Stdout= os.Stdout
	cmd.Stderr = os.Stdout
	cmd.Dir = "../../remoting/etcdv3/single"

	if err := cmd.Run(); err != nil{
		t.Fatal(err)
	}

}

func stopETCDServer(t *testing.T){

	cmd := exec.Command("./load.sh",  "stop")
	//cmd := exec.Command("pwd")
	cmd.Stdout= os.Stdout
	cmd.Stderr = os.Stdout
	cmd.Dir = "../../remoting/etcdv3/single"

	if err := cmd.Run(); err != nil{
		t.Fatal(err)
	}

}


func Test_DataChange(t *testing.T) {


	startETCDServer(t)
	defer stopETCDServer(t)

	listener := NewRegistryDataListener(&MockDataListener{})
	url, _ := common.NewURL(context.TODO(), "jsonrpc%3A%2F%2F127.0.0.1%3A20001%2Fcom.ikurento.user.UserProvider%3Fanyhost%3Dtrue%26app.version%3D0.0.1%26application%3DBDTService%26category%3Dproviders%26cluster%3Dfailover%26dubbo%3Ddubbo-provider-golang-2.6.0%26environment%3Ddev%26group%3D%26interface%3Dcom.ikurento.user.UserProvider%26ip%3D10.32.20.124%26loadbalance%3Drandom%26methods.GetUser.loadbalance%3Drandom%26methods.GetUser.retries%3D1%26methods.GetUser.weight%3D0%26module%3Ddubbogo%2Buser-info%2Bserver%26name%3DBDTService%26organization%3Dikurento.com%26owner%3DZX%26pid%3D74500%26retries%3D0%26service.filter%3Decho%26side%3Dprovider%26timestamp%3D1560155407%26version%3D%26warmup%3D100")
	listener.AddInterestedURL(&url)
	if !listener.DataChange(remoting.Event{Path: "/dubbo/com.ikurento.user.UserProvider/providers/jsonrpc%3A%2F%2F127.0.0.1%3A20001%2Fcom.ikurento.user.UserProvider%3Fanyhost%3Dtrue%26app.version%3D0.0.1%26application%3DBDTService%26category%3Dproviders%26cluster%3Dfailover%26dubbo%3Ddubbo-provider-golang-2.6.0%26environment%3Ddev%26group%3D%26interface%3Dcom.ikurento.user.UserProvider%26ip%3D10.32.20.124%26loadbalance%3Drandom%26methods.GetUser.loadbalance%3Drandom%26methods.GetUser.retries%3D1%26methods.GetUser.weight%3D0%26module%3Ddubbogo%2Buser-info%2Bserver%26name%3DBDTService%26organization%3Dikurento.com%26owner%3DZX%26pid%3D74500%26retries%3D0%26service.filter%3Decho%26side%3Dprovider%26timestamp%3D1560155407%26version%3D%26warmup%3D100"}) {
		t.Fatal("data change not ok")
	}
}

type MockDataListener struct {
}

func (*MockDataListener) Process(configType *remoting.ConfigChangeEvent) {

}
