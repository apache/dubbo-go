package config_center

import (
	"fmt"
)
import (
	"github.com/apache/dubbo-go/remoting"
)

type ConfigurationListener interface {
	Process(*ConfigChangeEvent)
}

type ConfigChangeEvent struct {
	Key        string
	Value      interface{}
	ConfigType remoting.EventType
}

func (c ConfigChangeEvent) String() string {
	return fmt.Sprintf("ConfigChangeEvent{key = %v , value = %v , changeType = %v}", c.Key, c.Value, c.ConfigType)
}
