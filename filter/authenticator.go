package filter

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/protocol"
)

// Authenticator
type Authenticator interface {

	// Sign
	// give a sign to request
	Sign(protocol.Invocation, *common.URL) error

	// Authenticate
	// verify the signature of the request is valid or not
	Authenticate(protocol.Invocation, *common.URL) error
}
