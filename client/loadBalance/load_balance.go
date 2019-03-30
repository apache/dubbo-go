package loadBalance

import (
	"github.com/dubbo/dubbo-go/client"
	"github.com/dubbo/dubbo-go/service"
)

type Selector interface {
	Select(ID int64, array client.ServiceArrayIf) (*service.ServiceURL, error)
}

//////////////////////////////////////////
// load balancer mode
//////////////////////////////////////////
//
//// Mode defines the algorithm of selecting a provider from cluster
//type Mode int
//
//const (
//	SM_BEGIN Mode = iota
//	SM_Random
//	SM_RoundRobin
//	SM_END
//)
//
//var modeStrings = [...]string{
//	"Begin",
//	"Random",
//	"RoundRobin",
//	"End",
//}
//
//func (s Mode) String() string {
//	if SM_BEGIN < s && s < SM_END {
//		return modeStrings[s]
//	}
//
//	return ""
//}
