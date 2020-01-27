package filter

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

type Authenticator interface {
	Sign(protocol.Invocation, *common.URL) error
	Authenticate(protocol.Invocation, *common.URL) error
}
