package directory

import (
	"github.com/dubbo/go-for-apache-dubbo/config"
	"github.com/dubbo/go-for-apache-dubbo/protocol"
)

type StaticDirectory struct {
	BaseDirectory
	invokers []protocol.Invoker
}

func NewStaticDirectory(invokers []protocol.Invoker) *StaticDirectory {
	return &StaticDirectory{
		BaseDirectory: NewBaseDirectory(&config.URL{}),
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

func (dir *StaticDirectory) Destroy() {
	dir.BaseDirectory.Destroy(func() {
		for _, ivk := range dir.invokers {
			ivk.Destroy()
		}
		dir.invokers = []protocol.Invoker{}
	})
}
