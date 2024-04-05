package engine

import (
	"dubbo.apache.org/dubbo-go/v3/istio/resources/rbac"
	envoyrbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
)

type RBACResult struct {
	ReqOK      bool
	PolicyName string
}

type RBACFilterEngine struct {
	RBAC          *envoyrbacv3.RBAC
	Authorization *rbac.Authorization
}

func NewRBACFilterEngine(rbac *envoyrbacv3.RBAC) *RBACFilterEngine {
	rbacFilterEngine := &RBACFilterEngine{
		RBAC: rbac,
	}
	return rbacFilterEngine
}

func (r *RBACFilterEngine) Filter(headers map[string]string) (*RBACResult, error) {
	rbacResult := &RBACResult{
		ReqOK: false,
	}
	if r.Authorization.Action == rbac.RBACLog {
		rbacResult.ReqOK = true
		return rbacResult, nil
	} else if r.Authorization.Action == rbac.RBACAllow {
		// when engine action is ALLOW, return allowed if matched any policy
		for name, policy := range r.Authorization.Policies {
			if policy.Match(headers) {
				rbacResult.PolicyName = name
				rbacResult.ReqOK = true
				return rbacResult, nil
			}
		}
		rbacResult.ReqOK = false
		return rbacResult, nil
	} else if r.Authorization.Action == rbac.RBACDeny {
		// when engine action is DENY, return allowed if not matched any policy
		for name, policy := range r.Authorization.Policies {
			if policy.Match(headers) {
				rbacResult.PolicyName = name
				rbacResult.ReqOK = false
				return rbacResult, nil
			}
		}
		rbacResult.ReqOK = true
		return rbacResult, nil
	}

	return rbacResult, nil
}
