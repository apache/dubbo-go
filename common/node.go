package common

import "github.com/dubbo/dubbo-go/config"

type Node interface {
	GetUrl() config.URL
	IsAvailable() bool
	Destroy()
}
