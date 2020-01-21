package filter_impl

import (
	"context"
	"errors"
	"strconv"
	"testing"
)

import (
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/apache/dubbo-go/protocol/mock"
)

func TestActiveFilter_Invoke(t *testing.T) {
	invoc := invocation.NewRPCInvocation("test", []interface{}{"OK"}, make(map[string]string, 0))
	url, _ := common.NewURL(context.TODO(), "dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	filter := ActiveFilter{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	invoker := mock.NewMockInvoker(ctrl)
	invoker.EXPECT().Invoke(gomock.Any()).Return(nil)
	invoker.EXPECT().GetUrl().Return(url).Times(1)
	filter.Invoke(context.Background(), invoker, invoc)
	assert.True(t, invoc.AttachmentsByKey(dubboInvokeStartTime, "") != "")

}

func TestActiveFilter_OnResponse(t *testing.T) {
	c := protocol.CurrentTimeMillis()
	elapsed := 100
	invoc := invocation.NewRPCInvocation("test", []interface{}{"OK"}, map[string]string{
		dubboInvokeStartTime: strconv.FormatInt(c-int64(elapsed), 10),
	})
	url, _ := common.NewURL(context.TODO(), "dubbo://192.168.10.10:20000/com.ikurento.user.UserProvider")
	filter := ActiveFilter{}
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	invoker := mock.NewMockInvoker(ctrl)
	invoker.EXPECT().GetUrl().Return(url).Times(1)
	result := &protocol.RPCResult{
		Err: errors.New("test"),
	}
	filter.OnResponse(nil, result, invoker, invoc)
	methodStatus := protocol.GetMethodStatus(url, "test")
	urlStatus := protocol.GetURLStatus(url)

	assert.Equal(t, int32(1), methodStatus.GetTotal())
	assert.Equal(t, int32(1), urlStatus.GetTotal())
	assert.Equal(t, int32(1), methodStatus.GetFailed())
	assert.Equal(t, int32(1), urlStatus.GetFailed())
	assert.Equal(t, int32(1), methodStatus.GetSuccessiveRequestFailureCount())
	assert.Equal(t, int32(1), urlStatus.GetSuccessiveRequestFailureCount())
	assert.True(t, methodStatus.GetFailedElapsed() >= int64(elapsed))
	assert.True(t, urlStatus.GetFailedElapsed() >= int64(elapsed))
	assert.True(t, urlStatus.GetLastRequestFailedTimestamp() != int64(0))
	assert.True(t, methodStatus.GetLastRequestFailedTimestamp() != int64(0))

}
