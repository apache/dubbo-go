package graceful_shutdown

import (
	"context"
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	"dubbo.apache.org/dubbo-go/v3/protocol/invocation"
	"github.com/stretchr/testify/assert"
	"net/url"
	"testing"
)

func TestConusmerFilterInvoke(t *testing.T) {
	url := common.NewURLWithOptions(common.WithParams(url.Values{}))
	invocation := invocation.NewRPCInvocation("GetUser", []interface{}{"OK"}, make(map[string]interface{}))

	rootConfig := config.NewRootConfigBuilder().
		SetShutDown(config.NewShutDownConfigBuilder().
			SetTimeout("60s").
			SetStepTimeout("3s").
			Build()).Build()

	config.SetRootConfig(*rootConfig)

	filter := extension.GetFilter(constant.GracefulShutdownConsumerFilterKey).(*consumerGracefulShutdownFilter)
	filter.Set(constant.GracefulShutdownFilterShutdownConfig, config.GetShutDown())

	assert.Equal(t, filter.shutdownConfig, config.GetShutDown())

	result := filter.Invoke(context.Background(), protocol.NewBaseInvoker(url), invocation)
	assert.NotNil(t, result)
	assert.Nil(t, result.Error())
}
