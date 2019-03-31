package selector

import (
	"fmt"
)

import (
	"github.com/dubbo/dubbo-go/client"
	"github.com/dubbo/dubbo-go/service"
)

var (
	ServiceArrayEmpty = fmt.Errorf("emtpy service array")
)

type Selector interface {
	Select(ID int64, array client.ServiceArrayIf) (*service.ServiceURL, error)
}
