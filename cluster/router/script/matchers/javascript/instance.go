package javascript

import (
	"github.com/dop251/goja"
	"go.uber.org/atomic"
	"sync"
)

var jsim jsInstanceManager
var once = sync.Once{}

type jsInstanceManager struct {
	insPool *sync.Pool
}

func newJsInstanceManager() jsInstanceManager {
	return jsInstanceManager{
		insPool: &sync.Pool{
			New: func() any {
				return newJsInstance()
			},
		},
	}
}

func GetInstanceManager() jsInstanceManager {
	once.Do(func() {
		jsim = newJsInstanceManager()
	})
	return jsim
}

type jsInstance struct {
	rt     *goja.Runtime
	enable atomic.Bool
}

func

func (it *jsInstance) Destroy() {
	it.enable.Swap(false)
	it.rt.Interrupt(nil)
}
func newJsInstance() *jsInstance {

}
