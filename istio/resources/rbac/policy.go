package rbac

import (
	rbacconfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"
)

type Policy struct {
	Permissions []Permission
	Principals  []Principal
}

func NewPolicy(rbacPolicy *rbacconfigv3.Policy) (*Policy, error) {
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
