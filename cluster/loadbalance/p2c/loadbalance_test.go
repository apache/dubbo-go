package p2c

import (
	"testing"
)

import (
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/cluster/metrics"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	protoinvoc "dubbo.apache.org/dubbo-go/v3/protocol/invocation"
)

func TestLoadBalance(t *testing.T) {
	lb := newP2CLoadBalance()
	invocation := protoinvoc.NewRPCInvocation("TestMethod", []interface{}{}, nil)
	randSeed = func() int64 {
		return 0
	}

	t.Run("no invokers", func(t *testing.T) {
		ivk := lb.Select([]protocol.Invoker{}, invocation)
		assert.Nil(t, ivk)
	})

	t.Run("one invoker", func(t *testing.T) {
		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")

		ivkArr := []protocol.Invoker{
			protocol.NewBaseInvoker(url0),
		}
		ivk := lb.Select(ivkArr, invocation)
		assert.Equal(t, ivkArr[0].GetURL().String(), ivk.GetURL().String())
	})

	t.Run("two invokers", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := metrics.NewMockMetrics(ctrl)
		metrics.LocalMetrics = m

		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
		url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url0), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(10), nil)
		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url1), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(5), nil)

		ivkArr := []protocol.Invoker{
			protocol.NewBaseInvoker(url0),
			protocol.NewBaseInvoker(url1),
		}

		ivk := lb.Select(ivkArr, invocation)

		assert.Equal(t, ivkArr[0].GetURL().String(), ivk.GetURL().String())
	})

	t.Run("multiple invokers", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := metrics.NewMockMetrics(ctrl)
		metrics.LocalMetrics = m

		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
		url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")
		url2, _ := common.NewURL("dubbo://192.168.1.2:20000/com.ikurento.user.UserProvider")

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url0), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(10), nil)
		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url1), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(5), nil)

		ivkArr := []protocol.Invoker{
			protocol.NewBaseInvoker(url0),
			protocol.NewBaseInvoker(url1),
			protocol.NewBaseInvoker(url2),
		}

		ivk := lb.Select(ivkArr, invocation)

		assert.Equal(t, ivkArr[0].GetURL().String(), ivk.GetURL().String())
	})

	t.Run("metrics i not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := metrics.NewMockMetrics(ctrl)
		metrics.LocalMetrics = m

		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
		url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")
		url2, _ := common.NewURL("dubbo://192.168.1.2:20000/com.ikurento.user.UserProvider")

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url0), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(0, metrics.ErrMetricsNotFound)

		ivkArr := []protocol.Invoker{
			protocol.NewBaseInvoker(url0),
			protocol.NewBaseInvoker(url1),
			protocol.NewBaseInvoker(url2),
		}

		ivk := lb.Select(ivkArr, invocation)

		assert.Equal(t, ivkArr[0].GetURL().String(), ivk.GetURL().String())
	})

	t.Run("metrics j not found", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		m := metrics.NewMockMetrics(ctrl)
		metrics.LocalMetrics = m

		url0, _ := common.NewURL("dubbo://192.168.1.0:20000/com.ikurento.user.UserProvider")
		url1, _ := common.NewURL("dubbo://192.168.1.1:20000/com.ikurento.user.UserProvider")
		url2, _ := common.NewURL("dubbo://192.168.1.2:20000/com.ikurento.user.UserProvider")

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url0), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(0), nil)

		m.EXPECT().
			GetMethodMetrics(gomock.Eq(url1), gomock.Eq(invocation.MethodName()), gomock.Eq(metrics.HillClimbing)).
			Times(1).
			Return(uint64(0), metrics.ErrMetricsNotFound)

		ivkArr := []protocol.Invoker{
			protocol.NewBaseInvoker(url0),
			protocol.NewBaseInvoker(url1),
			protocol.NewBaseInvoker(url2),
		}

		ivk := lb.Select(ivkArr, invocation)

		assert.Equal(t, ivkArr[1].GetURL().String(), ivk.GetURL().String())
	})

}
