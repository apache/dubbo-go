package etcdv3

import (
	"context"
	"strconv"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
)

func initRegistry(t *testing.T) *etcdV3Registry {

	regurl, err := common.NewURL(context.Background(), "registry://127.0.0.1:2379", common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	if err != nil {
		t.Fatal(err)
	}

	reg, err := newETCDV3Registry(&regurl)
	if err != nil {
		t.Fatal(err)
	}

	out := reg.(*etcdV3Registry)
	out.client.CleanKV()
	return out
}

func (suite *RegistryTestSuite) TestRegister() {

	t := suite.T()

	url, _ := common.NewURL(context.Background(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	reg := initRegistry(t)
	err := reg.Register(url)
	children, _, err := reg.client.GetChildrenKVList("/dubbo/com.ikurento.user.UserProvider/providers")
	if err != nil {
		t.Fatal(err)
	}
	assert.Regexp(t, ".*dubbo%3A%2F%2F127.0.0.1%3A20000%2Fcom.ikurento.user.UserProvider%3Fanyhost%3Dtrue%26category%3Dproviders%26cluster%3Dmock%26dubbo%3Ddubbo-provider-golang-2.6.0%26.*provider", children)
	assert.NoError(t, err)
}

func (suite *RegistryTestSuite) TestSubscribe() {

	t := suite.T()
	regurl, _ := common.NewURL(context.Background(), "registry://127.0.0.1:1111", common.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(common.PROVIDER)))
	url, _ := common.NewURL(context.Background(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	reg := initRegistry(t)
	//provider register
	err := reg.Register(url)
	if err != nil {
		t.Fatal(err)
	}

	//consumer register
	regurl.SetParam(constant.ROLE_KEY, strconv.Itoa(common.CONSUMER))
	reg2 := initRegistry(t)

	reg2.Register(url)
	listener, err := reg2.subscribe(&url)
	if err != nil {
		t.Fatal(err)
	}

	serviceEvent, err := listener.Next()
	if err != nil {
		t.Fatal(err)
	}
	assert.Regexp(t, ".*ServiceEvent{Action{add}.*", serviceEvent.String())
}

func (suite *RegistryTestSuite) TestConsumerDestory() {

	t := suite.T()
	url, _ := common.NewURL(context.Background(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))

	reg := initRegistry(t)
	_, err := reg.subscribe(&url)
	if err != nil {
		t.Fatal(err)
	}

	//listener.Close()
	time.Sleep(1e9)
	reg.Destroy()

	assert.Equal(t, false, reg.IsAvailable())

}

func (suite *RegistryTestSuite) TestProviderDestory() {

	t := suite.T()
	reg := initRegistry(t)
	url, _ := common.NewURL(context.Background(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", common.WithParamsValue(constant.CLUSTER_KEY, "mock"), common.WithMethods([]string{"GetUser", "AddUser"}))
	reg.Register(url)

	//listener.Close()
	time.Sleep(1e9)
	reg.Destroy()
	assert.Equal(t, false, reg.IsAvailable())
}
