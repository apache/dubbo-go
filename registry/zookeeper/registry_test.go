package zookeeper

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
	"github.com/dubbo/go-for-apache-dubbo/common/constant"
	"github.com/dubbo/go-for-apache-dubbo/config"
)

func Test_Register(t *testing.T) {
	regurl, _ := config.NewURL(context.TODO(), "registry://127.0.0.1:1111", config.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(config.PROVIDER)))
	url, _ := config.NewURL(context.TODO(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", config.WithParamsValue(constant.CLUSTER_KEY, "mock"), config.WithMethods([]string{"GetUser", "AddUser"}))

	ts, reg, err := NewMockZkRegistry(&regurl)
	defer ts.Stop()
	err = reg.Register(url)
	children, _ := reg.client.getChildren("/dubbo/com.ikurento.user.UserProvider/providers")
	assert.Regexp(t, ".*dubbo%3A%2F%2F127.0.0.1%3A20000%2Fcom.ikurento.user.UserProvider%3Fanyhost%3Dtrue%26category%3Dproviders%26cluster%3Dmock%26dubbo%3Ddubbo-provider-golang-2.6.0%26.*provider", children)
	assert.NoError(t, err)
}

func Test_Subscribe(t *testing.T) {
	regurl, _ := config.NewURL(context.TODO(), "registry://127.0.0.1:1111", config.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(config.PROVIDER)))
	url, _ := config.NewURL(context.TODO(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", config.WithParamsValue(constant.CLUSTER_KEY, "mock"), config.WithMethods([]string{"GetUser", "AddUser"}))
	ts, reg, err := NewMockZkRegistry(&regurl)
	defer ts.Stop()

	//provider register
	err = reg.Register(url)
	assert.NoError(t, err)

	if err != nil {
		return
	}

	//consumer register
	regurl.Params.Set(constant.ROLE_KEY, strconv.Itoa(config.CONSUMER))
	_, reg2, err := NewMockZkRegistry(&regurl)
	reg2.client = reg.client
	err = reg2.Register(url)
	listener, err := reg2.Subscribe(url)

	serviceEvent, err := listener.Next()
	assert.NoError(t, err)
	if err != nil {
		return
	}
	assert.Regexp(t, ".*ServiceEvent{Action{add service}.*", serviceEvent.String())

}

func Test_ConsumerDestory(t *testing.T) {
	regurl, _ := config.NewURL(context.TODO(), "registry://127.0.0.1:1111", config.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(config.CONSUMER)))
	url, _ := config.NewURL(context.TODO(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", config.WithParamsValue(constant.CLUSTER_KEY, "mock"), config.WithMethods([]string{"GetUser", "AddUser"}))

	ts, reg, err := NewMockZkRegistry(&regurl)
	defer ts.Stop()

	assert.NoError(t, err)
	err = reg.Register(url)
	assert.NoError(t, err)
	_, err = reg.Subscribe(url)
	assert.NoError(t, err)

	//listener.Close()
	time.Sleep(1e9)
	reg.Destroy()
	assert.Equal(t, false, reg.IsAvailable())

}

func Test_ProviderDestory(t *testing.T) {
	regurl, _ := config.NewURL(context.TODO(), "registry://127.0.0.1:1111", config.WithParamsValue(constant.ROLE_KEY, strconv.Itoa(config.PROVIDER)))
	url, _ := config.NewURL(context.TODO(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider", config.WithParamsValue(constant.CLUSTER_KEY, "mock"), config.WithMethods([]string{"GetUser", "AddUser"}))

	ts, reg, err := NewMockZkRegistry(&regurl)
	defer ts.Stop()

	assert.NoError(t, err)
	err = reg.Register(url)

	//listener.Close()
	time.Sleep(1e9)
	reg.Destroy()
	assert.Equal(t, false, reg.IsAvailable())
}
