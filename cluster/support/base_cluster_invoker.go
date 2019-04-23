package cluster

import (
	"context"
	"github.com/tevino/abool"
)

import (
	"github.com/dubbo/dubbo-go/cluster"
	"github.com/dubbo/dubbo-go/config"
)

type baseClusterInvoker struct {
	context        context.Context
	directory      cluster.Directory
	availablecheck bool
	destroyed      *abool.AtomicBool
}

func newBaseClusterInvoker(ctx context.Context, directory cluster.Directory) baseClusterInvoker {
	return baseClusterInvoker{
		context:        ctx,
		directory:      directory,
		availablecheck: false,
		destroyed:      abool.NewBool(false),
	}
}
func (invoker *baseClusterInvoker) GetUrl() config.IURL {
	return invoker.directory.GetUrl()
}

func (invoker *baseClusterInvoker) Destroy() {
	//this is must atom operation
	if invoker.destroyed.SetToIf(false, true) {
		invoker.directory.Destroy()
	}
}

func (invoker *baseClusterInvoker) IsAvailable() bool {
	//TODO:不理解java版本中关于stikyInvoker的逻辑所以先不写
	return invoker.directory.IsAvailable()
}
