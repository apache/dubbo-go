package rbac

import (
	"fmt"
	"reflect"

	rbacconfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
)

type Principal interface {
	isPrincipal()
	Match(headers map[string]string) bool
}

// Principal_Any
type PrincipalAny struct {
	Any bool
}

func (*PrincipalAny) isPrincipal() {}

func (principal *PrincipalAny) Match(headers map[string]string) bool {
	return principal.Any
}

func NewPrincipalAny(principal *rbacconfigv3.Principal_Any) (*PrincipalAny, error) {
	return &PrincipalAny{
		Any: principal.Any,
	}, nil
}

func NewPrincipal(principal *rbacconfigv3.Principal) (Principal, error) {

	switch principal.Identifier.(type) {
	case *rbacconfigv3.Principal_Any:
		return NewPrincipalAny(principal.Identifier.(*rbacconfigv3.Principal_Any))
	case *rbacconfigv3.Principal_DirectRemoteIp:
	case *rbacconfigv3.Principal_SourceIp:
	case *rbacconfigv3.Principal_RemoteIp:
	case *rbacconfigv3.Principal_Header:
	case *rbacconfigv3.Principal_AndIds:
	case *rbacconfigv3.Principal_OrIds:
	case *rbacconfigv3.Principal_NotId:
	case *rbacconfigv3.Principal_Metadata:
	default:
		return nil, fmt.Errorf("[NewPrincipal] not supported Principal.Identifier type found, detail: %v",
			reflect.TypeOf(principal.Identifier))
	}
	return nil, nil
}
