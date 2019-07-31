package etcdv3

import (
	"context"
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

import (
	"github.com/stretchr/testify/assert"
)

func initRegistry(t *testing.T) *etcdV3Registry {

	regurl, err := common.NewURL(context.TODO(), "registry://127.0.0.1:2379", common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	if err != nil {
		t.Fatal(err)
	}

	reg, err := newETCDV3Registry(&regurl)
	if err != nil {
		t.Fatal(err)
	}

	return reg.(*etcdV3Registry)
}

func Test_Register(t *testing.T) {
	url, _ := common.NewURL(context.TODO(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	reg := initRegistry(t)
	err := reg.Register(url)
	children, _, err := reg.client.GetChildrenKVList("/dubbo/com.ikurento.user.UserProvider/providers")
	if err != nil {
		t.Fatal(err)
	}
	assert.Regexp(t, ".*dubbo%3A%2F%2F127.0.0.1%3A20000%2Fcom.ikurento.user.UserProvider%3Fanyhost%3Dtrue%26category%3Dproviders%26cluster%3Dmock%26dubbo%3Ddubbo-provider-golang-2.6.0%26.*provider", children)
	assert.NoError(t, err)
}

func Test_Subscribe(t *testing.T) {

	regurl, _ := common.NewURL(context.TODO(), "registry://127.0.0.1:1111", common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	url, _ := common.NewURL(context.TODO(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	reg := initRegistry(t)
	//provider register
	err := reg.Register(url)
	if err != nil {
		t.Fatal(err)
	}

	//consumer register
	regurl.Params.Set(constant.ROLE_KEY, strconv.Itoa(common.CONSUMER))
	reg2 := initRegistry(t)

	reg2.Register(url)
	listener, err := reg2.Subscribe(url)
	if err != nil {
		t.Fatal(err)
	}

	serviceEvent, err := listener.Next()
	if err != nil {
		t.Fatal(err)
	}
	assert.Regexp(t, ".*ServiceEvent{Action{add}.*", serviceEvent.String())
}

func Test_ConsumerDestory(t *testing.T) {
	url, _ := common.NewURL(context.TODO(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	reg := initRegistry(t)
	_, err := reg.Subscribe(url)
	if err != nil {
		t.Fatal(err)
	}

	//listener.Close()
	time.Sleep(1e9)
	reg.Destroy()

	assert.Equal(t, false, reg.IsAvailable())

}

func Test_ProviderDestory(t *testing.T) {

	reg := initRegistry(t)
	url, _ := common.NewURL(context.TODO(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))
	reg.Register(url)

	//listener.Close()
	time.Sleep(1e9)
	reg.Destroy()
	assert.Equal(t, false, reg.IsAvailable())
}
