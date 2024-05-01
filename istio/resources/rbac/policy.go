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
	envoyrbacconfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
)

type Policy struct {
	Permissions []Permission
	Principals  []Principal
}

func NewPolicy(rbacPolicy *envoyrbacconfigv3.Policy) (*Policy, error) {
	policy := &Policy{}

	policy.Principals = make([]Principal, 0)
	policy.Permissions = make([]Permission, 0)

	for _, rbacPrincipal := range rbacPolicy.Principals {
		if principal, err := NewPrincipal(rbacPrincipal); err != nil {
			return nil, err
		} else {
			policy.Principals = append(policy.Principals, principal)
		}
	}

	for _, rbacPermission := range rbacPolicy.Permissions {
		if permission, err := NewPermission(rbacPermission); err != nil {
			return nil, err
		} else {
			policy.Permissions = append(policy.Permissions, permission)
		}
	}

	return policy, nil
}

func (p *Policy) Match(headers map[string]string) bool {
	permissionMatch, principalMatch := false, false
	for _, permission := range p.Permissions {
		if permission.Match(headers) {
			permissionMatch = true
			break
		}
	}

	for _, principal := range p.Principals {
		if principal.Match(headers) {
			principalMatch = true
			break
		}
	}

	return permissionMatch && principalMatch
}
