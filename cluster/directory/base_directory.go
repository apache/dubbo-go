package directory

import (
	"sync"
)
import (
	"github.com/tevino/abool"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/common"
)

type BaseDirectory struct {
	url       *common.URL
	destroyed *abool.AtomicBool
	mutex     sync.Mutex
}

func NewBaseDirectory(url *common.URL) BaseDirectory {
	return BaseDirectory{
		url:       url,
		destroyed: abool.NewBool(false),
	}
}
func (dir *BaseDirectory) GetUrl() common.URL {
	return *dir.url
}

func (dir *BaseDirectory) Destroy(doDestroy func()) {
	if dir.destroyed.SetToIf(false, true) {
		dir.mutex.Lock()
		doDestroy()
		dir.mutex.Unlock()
	}
}

func (dir *BaseDirectory) IsAvailable() bool {
	return !dir.destroyed.IsSet()
}
