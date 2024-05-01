/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package rbac

import (
	"fmt"
	"net"
	"reflect"
	"strconv"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
	envoyrbacconfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
)

// Permission define permission interface
type Permission interface {
	isPermission()
	Match(headers map[string]string) bool
}

type PermissionAny struct {
	// When any is set, it matches any action.
	Any bool
}

func (p *PermissionAny) isPermission() {}
func (p *PermissionAny) Match(headers map[string]string) bool {
	return p.Any
}

func NewPermissionAny(permission *envoyrbacconfigv3.Permission_Any) (*PermissionAny, error) {
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

func NewPermissionDestinationIp(permission *envoyrbacconfigv3.Permission_DestinationIp) (*PermissionDestinationIp, error) {
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

func NewPermissionDestinationPort(permission *envoyrbacconfigv3.Permission_DestinationPort) (*PermissionDestinationPort, error) {
	return &PermissionDestinationPort{
		DestinationPort: permission.DestinationPort,
	}, nil
}

type PermissionHeader struct {
	Matcher *HeaderMatcher
}

func NewPermissionHeader(permission *envoyrbacconfigv3.Permission_Header) (*PermissionHeader, error) {
	permissionHeader := &PermissionHeader{}
	if headerMatcher, err := NewHeaderMatcher(permission.Header); err != nil {
		return nil, err
	} else {
		permissionHeader.Matcher = headerMatcher
		return permissionHeader, nil
	}
}

func (p *PermissionHeader) isPermission() {}
func (p *PermissionHeader) Match(headers map[string]string) bool {
	return p.Matcher.Match(headers)
}

type PermissionAndRules struct {
	AndRules []Permission
}

func NewPermissionAndRules(permission *envoyrbacconfigv3.Permission_AndRules) (*PermissionAndRules, error) {
	permissionAndRules := &PermissionAndRules{}
	permissionAndRules.AndRules = make([]Permission, len(permission.AndRules.Rules))
	for idx, subPermission := range permission.AndRules.Rules {
		if subInheritPermission, err := NewPermission(subPermission); err != nil {
			return nil, err
		} else {
			permissionAndRules.AndRules[idx] = subInheritPermission
		}
	}
	return permissionAndRules, nil
}
func (p *PermissionAndRules) isPermission() {}
func (p *PermissionAndRules) Match(headers map[string]string) bool {
	for _, rule := range p.AndRules {
		if isMatch := rule.Match(headers); isMatch {
			continue
		} else {
			return false
		}
	}
	return true
}

type PermissionOrRules struct {
	OrRules []Permission
}

func NewPermissionOrRules(permission *envoyrbacconfigv3.Permission_OrRules) (*PermissionOrRules, error) {
	inheritPermission := &PermissionOrRules{}
	inheritPermission.OrRules = make([]Permission, len(permission.OrRules.Rules))
	for idx, subPermission := range permission.OrRules.Rules {
		if subInheritPermission, err := NewPermission(subPermission); err != nil {
			return nil, err
		} else {
			inheritPermission.OrRules[idx] = subInheritPermission
		}
	}
	return inheritPermission, nil
}
func (p *PermissionOrRules) isPermission() {}
func (p *PermissionOrRules) Match(headers map[string]string) bool {
	for _, rule := range p.OrRules {
		if isMatch := rule.Match(headers); isMatch {
			return true
		} else {
			continue
		}
	}
	return false
}

type PermissionNotRule struct {
	NotRule Permission
}

func NewPermissionNotRule(permission *envoyrbacconfigv3.Permission_NotRule) (*PermissionNotRule, error) {
	inheritPermission := &PermissionNotRule{}
	subPermission := permission.NotRule
	if subInheritPermission, err := NewPermission(subPermission); err != nil {
		return nil, err
	} else {
		inheritPermission.NotRule = subInheritPermission
	}
	return inheritPermission, nil
}

func (p *PermissionNotRule) isPermission() {}

func (p *PermissionNotRule) Match(headers map[string]string) bool {
	rule := p.NotRule
	return !rule.Match(headers)
}

type PermissionUrlPath struct {
	Matcher UrlPathMatcher
}

func NewPermissionUrlPath(permission *envoyrbacconfigv3.Permission_UrlPath) (*PermissionUrlPath, error) {
	inheritPermission := &PermissionUrlPath{}
	if urlPathMatcher, err := NewUrlPathMatcher(permission.UrlPath); err != nil {
		return nil, err
	} else {
		inheritPermission.Matcher = urlPathMatcher
		return inheritPermission, nil
	}
}
func (p *PermissionUrlPath) isPermission() {}

func (p *PermissionUrlPath) Match(headers map[string]string) bool {
	targetValue, found := getHeader(constant.HttpHeaderXPathName, headers)
	if !found {
		return false
	}
	return p.Matcher.Match(targetValue)
}

func NewPermission(permission *envoyrbacconfigv3.Permission) (Permission, error) {
	switch permission.Rule.(type) {
	case *envoyrbacconfigv3.Permission_Any:
		return NewPermissionAny(permission.Rule.(*envoyrbacconfigv3.Permission_Any))
	case *envoyrbacconfigv3.Permission_DestinationIp:
		return NewPermissionDestinationIp(permission.Rule.(*envoyrbacconfigv3.Permission_DestinationIp))
	case *envoyrbacconfigv3.Permission_DestinationPort:
		return NewPermissionDestinationPort(permission.Rule.(*envoyrbacconfigv3.Permission_DestinationPort))
	case *envoyrbacconfigv3.Permission_Header:
		return NewPermissionHeader(permission.Rule.(*envoyrbacconfigv3.Permission_Header))
	case *envoyrbacconfigv3.Permission_AndRules:
		return NewPermissionAndRules(permission.Rule.(*envoyrbacconfigv3.Permission_AndRules))
	case *envoyrbacconfigv3.Permission_OrRules:
		return NewPermissionOrRules(permission.Rule.(*envoyrbacconfigv3.Permission_OrRules))
	case *envoyrbacconfigv3.Permission_NotRule:
		return NewPermissionNotRule(permission.Rule.(*envoyrbacconfigv3.Permission_NotRule))
	case *envoyrbacconfigv3.Permission_UrlPath:
		return NewPermissionUrlPath(permission.Rule.(*envoyrbacconfigv3.Permission_UrlPath))
	default:
		return nil, fmt.Errorf("[NewPermission] not supported Permission.Rule type found, detail: %v",
			reflect.TypeOf(permission.Rule))
	}
}
