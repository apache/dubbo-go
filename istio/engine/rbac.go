package engine

import (
	envoyrbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
)

type RBACResult struct {
	ReqOK      bool
	PolicyName string
}

type RBACFilterEngine struct {
	RBAC *envoyrbacv3.RBAC
}

func NewRBACFilterEngine(rbac *envoyrbacv3.RBAC) *RBACFilterEngine {
	rbacFilterEngine := &RBACFilterEngine{
		RBAC: rbac,
	}
	return rbacFilterEngine
}

func (r *RBACFilterEngine) Filter(headers map[string]string) (*RBACResult, error) {
	return nil, nil
}
