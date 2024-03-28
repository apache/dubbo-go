package rbac

import (
	"fmt"
	"net"
	"reflect"
	"strconv"

	rbacconfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
)

// Permission define permission interface
type Permission interface {
	isPermission()
	Match(headers map[string]string) bool
}

type PermissionAny struct {
	Any bool
}

func (p *PermissionAny) isPermission() {}
func (p *PermissionAny) Match(headers map[string]string) bool {
	return true
}

func NewPermissionAny(permission *rbacconfigv3.Permission_Any) (*PermissionAny, error) {
	return &PermissionAny{
		Any: permission.Any,
	}, nil
}

type PermissionDestinationIp struct {
	CidrRange *net.IPNet
}

func (p *PermissionDestinationIp) isPermission() {}
func (p *PermissionDestinationIp) Match(headers map[string]string) bool {
	return true
}

func NewPermissionDestinationIp(permission *rbacconfigv3.Permission_DestinationIp) (*PermissionDestinationIp, error) {
	addressPrefix := permission.DestinationIp.AddressPrefix
	prefixLen := permission.DestinationIp.PrefixLen.GetValue()
	if _, ipNet, err := net.ParseCIDR(addressPrefix + "/" + strconv.Itoa(int(prefixLen))); err != nil {
		return nil, err
	} else {
		inheritPermission := &PermissionDestinationIp{
			CidrRange: ipNet,
		}
		return inheritPermission, nil
	}
}

type PermissionDestinationPort struct {
	DestinationPort uint32
}

func (p *PermissionDestinationPort) isPermission() {}
func (p *PermissionDestinationPort) Match(headers map[string]string) bool {
	return true
}

func NewPermissionDestinationPort(permission *rbacconfigv3.Permission_DestinationPort) (*PermissionDestinationPort, error) {
	return &PermissionDestinationPort{
		DestinationPort: permission.DestinationPort,
	}, nil
}

type PermissionHeader struct {
	Target      string
	Matcher     HeaderMatcher
	InvertMatch bool
}

func NewPermission(permission *rbacconfigv3.Permission) (Permission, error) {
	switch permission.Rule.(type) {
	case *rbacconfigv3.Permission_Any:
		return NewPermissionAny(permission.Rule.(*rbacconfigv3.Permission_Any))
	case *rbacconfigv3.Permission_DestinationIp:
		return NewPermissionDestinationIp(permission.Rule.(*rbacconfigv3.Permission_DestinationIp))
	case *rbacconfigv3.Permission_DestinationPort:
		return NewPermissionDestinationPort(permission.Rule.(*rbacconfigv3.Permission_DestinationPort))
	case *rbacconfigv3.Permission_Header:
	case *rbacconfigv3.Permission_AndRules:
	case *rbacconfigv3.Permission_OrRules:
	case *rbacconfigv3.Permission_NotRule:
	case *rbacconfigv3.Permission_UrlPath:
	default:
		return nil, fmt.Errorf("[NewPermission] not supported Permission.Rule type found, detail: %v",
			reflect.TypeOf(permission.Rule))
	}
	return nil, nil
}
