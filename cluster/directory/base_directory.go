package directory

import (
	"sync"
)
import (
	"go.uber.org/atomic"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/common"
)

type BaseDirectory struct {
	url       *common.URL
	destroyed *atomic.Bool
	mutex     sync.Mutex
}

func NewBaseDirectory(url *common.URL) BaseDirectory {
	return BaseDirectory{
		url:       url,
		destroyed: atomic.NewBool(false),
	}
}
func (dir *BaseDirectory) GetUrl() common.URL {
	return *dir.url
}

func (dir *BaseDirectory) Destroy(doDestroy func()) {
	if dir.destroyed.CAS(false, true) {
		dir.mutex.Lock()
		doDestroy()
		dir.mutex.Unlock()
	}
}

func (dir *BaseDirectory) IsAvailable() bool {
	return !dir.destroyed.Load()
}
