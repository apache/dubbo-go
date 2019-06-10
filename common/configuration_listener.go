package common

import "fmt"

type ConfigurationListener interface {
	Process(*ConfigChangeEvent)
}

type ConfigChangeEvent struct {
	Key        string
	Value      interface{}
	ConfigType EventType
}

func (c ConfigChangeEvent) String() string {
	return fmt.Sprintf("ConfigChangeEvent{key = %v , value = %v , changeType = %v}", c.Key, c.Value, c.ConfigType)
}

//////////////////////////////////////////
// event type
//////////////////////////////////////////

type EventType int

const (
	Add = iota
	Del
)

var serviceEventTypeStrings = [...]string{
	"add",
	"delete",
}

func (t EventType) String() string {
	return serviceEventTypeStrings[t]
}

//////////////////////////////////////////
// service event
//////////////////////////////////////////

type Event struct {
	Path    string
	Action  EventType
	Content string
}

func (e Event) String() string {
	return fmt.Sprintf("Event{Action{%s}, Content{%s}}", e.Action, e.Content)
}
