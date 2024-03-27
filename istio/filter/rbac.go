package filter

import (
	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
)

type RBACResult struct {
}

type RBACFilterEngine struct {
	headers map[string]string
	RBAC    *rbacv3.RBAC
}

func NewRBACFilterEngine(headers map[string]string, rbac *rbacv3.RBAC) *RBACFilterEngine {
	rbacFilterEngine := &RBACFilterEngine{
		headers: headers,
		RBAC:    rbac,
	}
	return rbacFilterEngine
}

func (r *RBACFilterEngine) Filter() (*RBACResult, error) {

	return nil, nil
}