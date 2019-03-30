package registry

import (
	"fmt"
	"math/rand"
	"time"
)

import (
	"github.com/dubbo/dubbo-go/service"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

//////////////////////////////////////////
// service url event type
//////////////////////////////////////////

type ServiceURLEventType int

const (
	ServiceURLAdd = iota
	ServiceURLDel
)

var serviceURLEventTypeStrings = [...]string{
	"add service url",
	"delete service url",
}

func (t ServiceURLEventType) String() string {
	return serviceURLEventTypeStrings[t]
}

//////////////////////////////////////////
// service url event
//////////////////////////////////////////

type ServiceURLEvent struct {
	Action  ServiceURLEventType
	Service *service.ServiceURL
}

func (e ServiceURLEvent) String() string {
	return fmt.Sprintf("ServiceURLEvent{Action{%s}, Service{%s}}", e.Action.String(), e.Service)
}
