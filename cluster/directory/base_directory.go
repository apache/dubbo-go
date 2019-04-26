package directory

import (
	"github.com/dubbo/dubbo-go/config"
)

type BaseDirectory struct {
	url       *config.RegistryURL
	destroyed bool
}

func NewBaseDirectory(url *config.RegistryURL) BaseDirectory {
	return BaseDirectory{
		url: url,
	}
}
func (dir *BaseDirectory) GetUrl() config.IURL {
	return dir.url
}

func (dir *BaseDirectory) Destroy() {
	dir.destroyed = false
}
