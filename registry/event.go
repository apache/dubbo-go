package registry

import (
	"fmt"
	"math/rand"
	"time"
)
import (
	"github.com/dubbo/dubbo-go/config"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

//////////////////////////////////////////
// service url event type
//////////////////////////////////////////

type ServiceEventType int

const (
	ServiceAdd = iota
	ServiceDel
)

var serviceEventTypeStrings = [...]string{
	"add service",
	"delete service",
}

func (t ServiceEventType) String() string {
	return serviceEventTypeStrings[t]
}

//////////////////////////////////////////
// service event
//////////////////////////////////////////

type ServiceEvent struct {
	Action  ServiceEventType
	Service config.URL
}

func (e ServiceEvent) String() string {
	return fmt.Sprintf("ServiceEvent{Action{%s}, Service{%s}}", e.Action, e.Service)
}
