package directory

import (
	"github.com/dubbo/dubbo-go/protocol"
)

type StaticDirectory struct {
	BaseDirectory
	invokers []protocol.Invoker
}

func NewStaticDirectory(invokers []protocol.Invoker) *StaticDirectory {
	return &StaticDirectory{
		BaseDirectory: NewBaseDirectory(nil),
		invokers:      invokers,
	}
}

//for-loop invokers ,if all invokers is available ,then it means directory is available
func (dir *StaticDirectory) IsAvailable() bool {
	for _, invoker := range dir.invokers {
		if !invoker.IsAvailable() {
			return false
		}
	}
	return true
}

func (dir *StaticDirectory) List(invocation protocol.Invocation) []protocol.Invoker {
	//TODO:Here should add router
	return dir.invokers
}
