package directory

import (
	"github.com/dubbo/go-for-apache-dubbo/common"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type staticDirectory struct {
	BaseDirectory
	invokers []protocol.Invoker
}

func NewStaticDirectory(invokers []protocol.Invoker) *staticDirectory {
	return &staticDirectory{
		BaseDirectory: NewBaseDirectory(&common.URL{}),
		invokers:      invokers,
	}
}

//for-loop invokers ,if all invokers is available ,then it means directory is available
func (dir *staticDirectory) IsAvailable() bool {
	for _, invoker := range dir.invokers {
		if !invoker.IsAvailable() {
			return false
		}
	}
	return true
}

func (dir *staticDirectory) List(invocation protocol.Invocation) []protocol.Invoker {
	//TODO:Here should add router
	return dir.invokers
}

func (dir *staticDirectory) Destroy() {
	dir.BaseDirectory.Destroy(func() {
		for _, ivk := range dir.invokers {
			ivk.Destroy()
		}
		dir.invokers = []protocol.Invoker{}
	})
}
