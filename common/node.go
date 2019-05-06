package common

import "github.com/dubbo/go-for-apache-dubbo/config"

type Node interface {
	GetUrl() config.URL
	IsAvailable() bool
	Destroy()
}
