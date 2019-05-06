package directory

import (
	"github.com/tevino/abool"
)
import (
	"github.com/dubbo/go-for-apache-dubbo/config"
)

type BaseDirectory struct {
	url       *config.URL
	destroyed *abool.AtomicBool
}

func NewBaseDirectory(url *config.URL) BaseDirectory {
	return BaseDirectory{
		url:       url,
		destroyed: abool.NewBool(false),
	}
}
func (dir *BaseDirectory) GetUrl() config.URL {
	return *dir.url
}

func (dir *BaseDirectory) Destroy() {
	if dir.destroyed.SetToIf(false, true) {
	}
}

func (dir *BaseDirectory) IsAvailable() bool {
	return !dir.destroyed.IsSet()
}
