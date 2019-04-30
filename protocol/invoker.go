package protocol

import (
	log "github.com/AlexStocks/log4go"
)

import (
	"github.com/dubbo/dubbo-go/common"
	"github.com/dubbo/dubbo-go/config"
)

// Extension - Invoker
type Invoker interface {
	common.Node
	Invoke(Invocation) Result
}

/////////////////////////////
// base invoker
/////////////////////////////

type BaseInvoker struct {
	url       *config.URL
	available bool
	destroyed bool
}

func NewBaseInvoker(url *config.URL) *BaseInvoker {
	return &BaseInvoker{
		url:       url,
		available: true,
		destroyed: false,
	}
}

func (bi *BaseInvoker) GetUrl() config.URL {
	return *bi.url
}

func (bi *BaseInvoker) IsAvailable() bool {
	return bi.available
}

func (bi *BaseInvoker) IsDestroyed() bool {
	return bi.destroyed
}

func (bi *BaseInvoker) Invoke(invocation Invocation) Result {
	return &RPCResult{}
}

func (bi *BaseInvoker) Destroy() {
	log.Info("Destroy invoker: %s", bi.GetUrl().String())
	bi.destroyed = true
	bi.available = false
}
