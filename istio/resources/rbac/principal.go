package rbac

import (
	"fmt"
	"net"
	"reflect"
	"strconv"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	envoyrbacconfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
	envoymatcherv3 "github.com/envoyproxy/go-control-plane/envoy/type/matcher/v3"
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
	ip := headers[constant.HttpHeaderXRemoteIp]
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
	ip := headers[constant.HttpHeaderXRemoteIp]
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
	Matcher     HeaderMatcher
	InvertMatch bool
}

func (p *PrincipalHeader) isPrincipal() {}

func (p *PrincipalHeader) Match(headers map[string]string) bool {
	targetValue, found := getHeader(p.Target, headers)

	// HeaderMatcherPresentMatch is a little special
	if matcher, ok := p.Matcher.(*HeaderMatcherPresentMatch); ok {
		// HeaderMatcherPresentMatch matches if and only if header found and PresentMatch is true
		isMatch := found && matcher.PresentMatch
		return p.InvertMatch != isMatch
	}

	// return false when targetValue is not found, except matcher is `HeaderMatcherPresentMatch`
	if !found {
		return false
	} else {
		isMatch := p.Matcher.Match(targetValue)
		// principal.InvertMatch xor isMatch
		return p.InvertMatch != isMatch
	}
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
	Path    string
	Matcher StringMatcher
}

func NewPrincipalMetadata(principal *envoyrbacconfigv3.Principal_Metadata) (*PrincipalMetadata, error) {
	if principal.Metadata == nil || principal.Metadata.Value == nil {
		return nil, fmt.Errorf("unsupported Principal_Metadata Metadata nil: %v", principal)
	}
	if principal.Metadata.Filter != "istio_authn" {
		return nil, fmt.Errorf("unsupported Principal_Metadata filter: %s", principal.Metadata.Filter)
	}
	path := principal.Metadata.Path
	if len(path) == 0 || path[0].GetKey() != "source.principal" {
		return nil, fmt.Errorf("unsupported Principal_Metadata path: %v", path)
	}
	matcher, ok := principal.Metadata.Value.MatchPattern.(*envoymatcherv3.ValueMatcher_StringMatch)
	if !ok {
		return nil, fmt.Errorf("unsupported Principal_Metadata matcher: %s", reflect.TypeOf(principal.Metadata.Value))
	}
	stringMatcher, err := NewStringMatcher(matcher.StringMatch)
	if err != nil {
		return nil, err
	}

	return &PrincipalMetadata{
		Filter:  principal.Metadata.Filter,
		Path:    path[0].GetKey(),
		Matcher: stringMatcher,
	}, nil
}
func (p *PrincipalMetadata) isPrincipal() {}

func (p *PrincipalMetadata) Match(header map[string]string) bool {
	if p.Filter != "istio_authn" {
		return false
	}
	if p.Path != "source.principal" {
		return false
	}
	//todo check principal
	return true
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
	default:
		return nil, fmt.Errorf("[NewPrincipal] not supported Principal.Identifier type found, detail: %v",
			reflect.TypeOf(principal.Identifier))
	}
	return nil, nil
}
