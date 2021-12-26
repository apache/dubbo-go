package graceful_shutdown

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/filter"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	perrors "github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

func TestProviderFilterInvoke(t *testing.T) {
	url := common.NewURLWithOptions(common.WithParams(url.Values{}))
	invocation := invocation.NewRPCInvocation("GetUser", []interface{}{"OK"}, make(map[string]interface{}))

	extension.SetRejectedExecutionHandler("test", func() filter.RejectedExecutionHandler {
		return &TestRejectedExecutionHandler{}
	})

	rootConfig := config.NewRootConfigBuilder().
		SetShutDown(config.NewShutDownConfigBuilder().
			SetTimeout("60s").
			SetStepTimeout("3s").
			SetRejectRequestHandler("test").
			Build()).Build()

	config.SetRootConfig(*rootConfig)

	filter := extension.GetFilter(constant.GracefulShutdownProviderFilterKey).(*providerGracefulShutdownFilter)
	filter.Set(constant.GracefulShutdownFilterShutdownConfig, config.GetShutDown())
	assert.Equal(t, filter.shutdownConfig, config.GetShutDown())

	result := filter.Invoke(context.Background(), protocol.NewBaseInvoker(url), invocation)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())

	config.GetShutDown().RejectRequest = true
	result = filter.Invoke(context.Background(), protocol.NewBaseInvoker(url), invocation)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Error().Error(), "Rejected")

}

type TestRejectedExecutionHandler struct{}

// RejectedExecution will do nothing, it only log the invocation.
func (handler *TestRejectedExecutionHandler) RejectedExecution(url *common.URL,
	_ protocol.Invocation) protocol.Result {

	return &protocol.RPCResult{
		Err: perrors.New("Rejected"),
	}
}
