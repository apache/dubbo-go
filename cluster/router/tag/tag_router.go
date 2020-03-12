package tag

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/protocol"
	perrors "github.com/pkg/errors"
)

type tagRouter struct {
	url     *common.URL
	enabled bool
}

func NewTagRouter(url *common.URL) (*tagRouter, error) {
	if url == nil {
		return nil, perrors.Errorf("Illegal route URL!")
	}
	return &tagRouter{
		url:     url,
		enabled: url.GetParamBool(constant.RouterEnabled, true),
	}, nil
}

func (c *tagRouter) isEnabled() bool {
	return c.enabled
}

func (c *tagRouter) Route(invokers []protocol.Invoker, url *common.URL, invocation protocol.Invocation) []protocol.Invoker {
	if !c.isEnabled() {
		return invokers
	}
	if len(invokers) == 0 {
		return invokers
	}
	// todo
}

func (c *tagRouter) URL() common.URL {
	return *c.url
}
func (c *tagRouter) Priority() int64 {
	// todo
}
