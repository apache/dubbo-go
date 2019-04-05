package selector

import (
	"fmt"
)

import (
	"github.com/dubbo/dubbo-go/client"
	"github.com/dubbo/dubbo-go/registry"
)

var (
	ServiceArrayEmpty = fmt.Errorf("emtpy service array")
)

type Selector interface {
	Select(ID int64, array client.ServiceArrayIf) (registry.ServiceURL, error)
}
