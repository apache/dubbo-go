package engine

import (
	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
)

type RBACResult struct {
}

type RBACFilterEngine struct {
	RBAC *rbacv3.RBAC
}

func NewRBACFilterEngine(rbac *rbacv3.RBAC) *RBACFilterEngine {
	rbacFilterEngine := &RBACFilterEngine{
		RBAC: rbac,
	}
	return rbacFilterEngine
}

func (r *RBACFilterEngine) Filter(headers map[string]string) (*RBACResult, error) {

	return nil, nil
}
