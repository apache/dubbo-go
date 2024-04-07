package rbac

import (
	"fmt"
	"net"
	"reflect"
	"strconv"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	envoyrbacconfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
)

type Principal interface {
	isPrincipal()
	Match(headers map[string]string) bool
}

type PrincipalAny struct {
	Any bool
}

func (p *PrincipalAny) isPrincipal() {}

func (p *PrincipalAny) Match(headers map[string]string) bool {
	return p.Any
}

func NewPrincipalAny(principal *envoyrbacconfigv3.Principal_Any) (*PrincipalAny, error) {
	return &PrincipalAny{
		Any: principal.Any,
	}, nil
}

type PrincipalDirectRemoteIp struct {
	CidrRange *net.IPNet
}

func (p *PrincipalDirectRemoteIp) isPrincipal() {}

func (p *PrincipalDirectRemoteIp) Match(headers map[string]string) bool {
	ip := headers[constant.HttpHeaderXSourceIp]
	if len(ip) == 0 {
		return false
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		// Handle invalid IP address case (log error, return false, etc.)
		return false
	}
	if !p.CidrRange.Contains(parsedIP) {
		return false
	}
	return true
}

func NewPrincipalDirectRemoteIp(principal *envoyrbacconfigv3.Principal_DirectRemoteIp) (*PrincipalDirectRemoteIp, error) {
	addressPrefix := principal.DirectRemoteIp.AddressPrefix
	prefixLen := principal.DirectRemoteIp.PrefixLen.GetValue()
	if _, ipNet, err := net.ParseCIDR(addressPrefix + "/" + strconv.Itoa(int(prefixLen))); err != nil {
		return nil, err
	} else {
		inheritPrincipal := &PrincipalDirectRemoteIp{
			CidrRange: ipNet,
		}
		return inheritPrincipal, nil
	}
}

// Deprecated: Do not use.
type PrincipalSourceIp struct {
	CidrRange *net.IPNet
}

func (p *PrincipalSourceIp) isPrincipal() {}

func (p *PrincipalSourceIp) Match(headers map[string]string) bool {
	ip := headers[constant.HttpHeaderXSourceIp]
	if len(ip) == 0 {
		return false
	}
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		// Handle invalid IP address case (log error, return false, etc.)
		return false
	}
	if !p.CidrRange.Contains(parsedIP) {
		return false
	}
	return true
}

func NewPrincipalSourceIp(principal *envoyrbacconfigv3.Principal_SourceIp) (*PrincipalSourceIp, error) {
	addressPrefix := principal.SourceIp.AddressPrefix
	prefixLen := principal.SourceIp.PrefixLen.GetValue()
	if _, ipNet, err := net.ParseCIDR(addressPrefix + "/" + strconv.Itoa(int(prefixLen))); err != nil {
		return nil, err
	} else {
		inheritPrincipal := &PrincipalSourceIp{
			CidrRange: ipNet,
		}
		return inheritPrincipal, nil
	}
}

type PrincipalHeader struct {
	Target      string
	Matcher     *HeaderMatcher
	InvertMatch bool
}

func (p *PrincipalHeader) isPrincipal() {}

func (p *PrincipalHeader) Match(headers map[string]string) bool {
	return p.Matcher.Match(headers)
}

func NewPrincipalHeader(principal *envoyrbacconfigv3.Principal_Header) (*PrincipalHeader, error) {
	principalHeader := &PrincipalHeader{}
	principalHeader.Target = principal.Header.Name
	principalHeader.InvertMatch = principal.Header.InvertMatch
	if headerMatcher, err := NewHeaderMatcher(principal.Header); err != nil {
		return nil, err
	} else {
		principalHeader.Matcher = headerMatcher
		return principalHeader, nil
	}
}

type PrincipalAndIds struct {
	AndIds []Principal
}

func (p *PrincipalAndIds) isPrincipal() {}

func (p *PrincipalAndIds) Match(headers map[string]string) bool {
	for _, ids := range p.AndIds {
		if isMatch := ids.Match(headers); isMatch {
			continue
		} else {
			return false
		}
	}
	return true
}

func NewPrincipalAndIds(principal *envoyrbacconfigv3.Principal_AndIds) (*PrincipalAndIds, error) {
	principalIds := &PrincipalAndIds{}
	principalIds.AndIds = make([]Principal, len(principal.AndIds.Ids))
	for idx, subPrincipal := range principal.AndIds.Ids {
		if subInheritPrincipal, err := NewPrincipal(subPrincipal); err != nil {
			return nil, err
		} else {
			principalIds.AndIds[idx] = subInheritPrincipal
		}
	}
	return principalIds, nil
}

type PrincipalOrIds struct {
	OrIds []Principal
}

func NewPrincipalOrIds(principal *envoyrbacconfigv3.Principal_OrIds) (*PrincipalOrIds, error) {
	principalOrIds := &PrincipalOrIds{}
	principalOrIds.OrIds = make([]Principal, len(principal.OrIds.Ids))
	for idx, subPrincipal := range principal.OrIds.Ids {
		if subInheritPrincipal, err := NewPrincipal(subPrincipal); err != nil {
			return nil, err
		} else {
			principalOrIds.OrIds[idx] = subInheritPrincipal
		}
	}
	return principalOrIds, nil
}

func (p *PrincipalOrIds) isPrincipal() {}

func (p *PrincipalOrIds) Match(headers map[string]string) bool {
	for _, ids := range p.OrIds {
		if isMatch := ids.Match(headers); isMatch {
			return true
		} else {
			continue
		}
	}
	return false
}

type PrincipalNotId struct {
	NotId Principal
}

func NewPrincipalNotId(principal *envoyrbacconfigv3.Principal_NotId) (*PrincipalNotId, error) {
	principalNotId := &PrincipalNotId{}
	subPermission := principal.NotId
	if subInheritPermission, err := NewPrincipal(subPermission); err != nil {
		return nil, err
	} else {
		principalNotId.NotId = subInheritPermission
	}
	return principalNotId, nil
}

func (p *PrincipalNotId) isPrincipal() {}

func (p *PrincipalNotId) Match(headers map[string]string) bool {
	rule := p.NotId
	return !rule.Match(headers)
}

type PrincipalMetadata struct {
	Filter  string
	Path    []string
	Matcher *ValueMatcher
	Invert  bool
}

func NewPrincipalMetadata(principal *envoyrbacconfigv3.Principal_Metadata) (*PrincipalMetadata, error) {
	if principal.Metadata == nil || principal.Metadata.Value == nil {
		return nil, fmt.Errorf("unsupported Principal_Metadata Metadata nil: %v", principal)
	}
	if principal.Metadata.Filter != "istio_authn" {
		return nil, fmt.Errorf("unsupported Principal_Metadata filter: %s", principal.Metadata.Filter)
	}
	filter := principal.Metadata.Filter
	path := make([]string, 0)
	for _, pathSegment := range principal.Metadata.Path {
		if len(pathSegment.GetKey()) > 0 {
			path = append(path, pathSegment.GetKey())
		}
	}
	invert := principal.Metadata.Invert
	value, err := NewValueMatcher(principal.Metadata.Value)
	if err != nil {
		return nil, err
	}

	return &PrincipalMetadata{
		Filter:  filter,
		Path:    path,
		Matcher: value,
		Invert:  invert,
	}, nil
}
func (p *PrincipalMetadata) isPrincipal() {}

func (p *PrincipalMetadata) Match(headers map[string]string) bool {
	if p.Filter != "istio_authn" {
		return false
	}
	if len(p.Path) == 0 {
		return false
	}
	headerKey := strings.Join(p.Path, ".")
	targetValue, ok := getHeader(headerKey, headers)
	if !ok {
		return false
	}
	return p.Matcher.Match(targetValue)
}

type PrincipalAuthenticated struct {
	PrincipalName *StringMatcher
}

func (p *PrincipalAuthenticated) isPrincipal() {}

func (p *PrincipalAuthenticated) Match(headers map[string]string) bool {
	sourcePrincipal, ok := getHeader(constant.HttpHeaderXSourcePrincipal, headers)
	if !ok {
		return false
	}
	return p.PrincipalName.Match(sourcePrincipal)
}

func NewPrincipalAuthenticated(principal *envoyrbacconfigv3.Principal_Authenticated_) (*PrincipalAuthenticated, error) {
	stringMatcher, err := NewStringMatcher(principal.Authenticated.PrincipalName)
	if err != nil {
		return nil, err
	}
	return &PrincipalAuthenticated{
		PrincipalName: stringMatcher,
	}, nil
}

func NewPrincipal(principal *envoyrbacconfigv3.Principal) (Principal, error) {

	switch principal.Identifier.(type) {
	case *envoyrbacconfigv3.Principal_Any:
		return NewPrincipalAny(principal.Identifier.(*envoyrbacconfigv3.Principal_Any))
	case *envoyrbacconfigv3.Principal_DirectRemoteIp:
		return NewPrincipalDirectRemoteIp(principal.Identifier.(*envoyrbacconfigv3.Principal_DirectRemoteIp))
	case *envoyrbacconfigv3.Principal_SourceIp:
		return NewPrincipalSourceIp(principal.Identifier.(*envoyrbacconfigv3.Principal_SourceIp))
	case *envoyrbacconfigv3.Principal_RemoteIp:
		// it is same as source ip
		return NewPrincipalSourceIp(principal.Identifier.(*envoyrbacconfigv3.Principal_SourceIp))
	case *envoyrbacconfigv3.Principal_Header:
		return NewPrincipalHeader(principal.Identifier.(*envoyrbacconfigv3.Principal_Header))
	case *envoyrbacconfigv3.Principal_AndIds:
		return NewPrincipalAndIds(principal.Identifier.(*envoyrbacconfigv3.Principal_AndIds))
	case *envoyrbacconfigv3.Principal_OrIds:
		return NewPrincipalOrIds(principal.Identifier.(*envoyrbacconfigv3.Principal_OrIds))
	case *envoyrbacconfigv3.Principal_NotId:
		return NewPrincipalNotId(principal.Identifier.(*envoyrbacconfigv3.Principal_NotId))
	case *envoyrbacconfigv3.Principal_Metadata:
		return NewPrincipalMetadata(principal.Identifier.(*envoyrbacconfigv3.Principal_Metadata))
	case *envoyrbacconfigv3.Principal_Authenticated_:
		return NewPrincipalAuthenticated(principal.Identifier.(*envoyrbacconfigv3.Principal_Authenticated_))
	default:
		return nil, fmt.Errorf("[NewPrincipal] not supported Principal.Identifier type found, detail: %v",
			reflect.TypeOf(principal.Identifier))
	}
	return nil, nil
}
