package registry

import (
	"fmt"
	"math/rand"
	"time"
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
	Service *DefaultServiceURL
}

func (e ServiceEvent) String() string {
	return fmt.Sprintf("ServiceEvent{Action{%s}, Service{%s}}", e.Action, e.Service)
}
