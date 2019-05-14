package common

type Node interface {
	GetUrl() URL
	IsAvailable() bool
	Destroy()
}
