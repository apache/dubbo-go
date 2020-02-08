package auth

import (
	"context"
	"testing"
)

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/mock"
)

func TestConsumerSignFilter_Invoke(t *testing.T) {
	url, _ := common.NewURL(context.TODO(), "dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=gg&version=2.6.0")
	url.SetParam(constant.SECRET_ACCESS_KEY_KEY, "sk")
	url.SetParam(constant.ACCESS_KEY_ID_KEY, "ak")
	inv := invocation.NewRPCInvocation("test", []interface{}{"OK"}, nil)
	filter := &ConsumerSignFilter{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	invoker := mock.NewMockInvoker(ctrl)
	result := &protocol.RPCResult{}
	invoker.EXPECT().Invoke(inv).Return(result).Times(2)
	invoker.EXPECT().GetUrl().Return(url).Times(2)
	assert.Equal(t, result, filter.Invoke(context.Background(), invoker, inv))

	url.SetParam(constant.SERVICE_AUTH_KEY, "true")
	assert.Equal(t, result, filter.Invoke(context.Background(), invoker, inv))
}
