package channel

import (
	"testing"
)

func TestSdsClientChannel(t *testing.T) {
	//udsPath := "/Users/jun/GolandProjects/dubbo/dubbo-mesh/var/run/dubbomesh/workload-spiffe-uds/socket"
	//node := &v3configcore.Node{
	//	Id: "sidecar~10.10.241.72~.~.svc.cluster.local",
	//}
	//listener := func(secret *tls.Secret) error {
	//	logger.Infof("recv secret:%v", secret)
	//	return nil
	//}
	//
	//sdsClientChannel, _ := NewSdsClientChannel(udsPath, node)
	//sdsClientChannel.AddListener(listener, "sds")
	//sdsClientChannel.Send([]string{"default", "ROOTCA"})
	//
	//select {
	//case <-time.After(60 * time.Second):
	//	logger.Infof("dealy 30 seconds")
	//}
	//
	//sdsClientChannel.Stop()
}
