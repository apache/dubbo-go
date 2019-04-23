package cluster

import "github.com/dubbo/dubbo-go/common"

// Extension - Directory
type Directory interface {
	common.Node
	List()
}
