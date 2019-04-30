package directory

import (
	"github.com/dubbo/dubbo-go/config"
)

type BaseDirectory struct {
	url       *config.URL
	destroyed bool
}

func NewBaseDirectory(url *config.URL) BaseDirectory {
	return BaseDirectory{
		url: url,
	}
}
func (dir *BaseDirectory) GetUrl() config.URL {
	return *dir.url
}

func (dir *BaseDirectory) Destroy() {
	dir.destroyed = false
}
