package directory

import (
	"context"
	"github.com/dubbo/dubbo-go/config"
)

type BaseDirectory struct {
	context   context.Context
	url       *config.RegistryURL
	destroyed bool
}

func NewBaseDirectory(ctx context.Context, url *config.RegistryURL) BaseDirectory {
	return BaseDirectory{
		context: ctx,
		url:     url,
	}
}
func (dir *BaseDirectory) GetUrl() config.IURL {
	return dir.url
}

func (dir *BaseDirectory) Destroy() {
	dir.destroyed = false
}

func (dir *BaseDirectory) Context() context.Context {
	return dir.context
}
