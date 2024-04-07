package engine

import (
	"dubbo.apache.org/dubbo-go/v3/istio/resources/rbac"
)

type RBACResult struct {
	ReqOK           bool
	MatchPolicyName string
}

type RBACFilterEngine struct {
	RBAC *rbac.RBAC
}

func NewRBACFilterEngine(rbac *rbac.RBAC) *RBACFilterEngine {
	rbacFilterEngine := &RBACFilterEngine{
		RBAC: rbac,
	}
	return rbacFilterEngine
}

func (r *RBACFilterEngine) Filter(headers map[string]string) (*RBACResult, error) {
	var allowed bool
	var matchPolicyName string
	var err error

	// rbac shadow rules handle
	if r.RBAC.ShadowRules != nil {
		allowed, matchPolicyName, err = r.RBAC.ShadowRules.Match(headers)
	}

	// rbac rules handle
	if r.RBAC.Rules != nil {
		allowed, matchPolicyName, err = r.RBAC.Rules.Match(headers)
	} else {
		allowed = true
		err = nil
	}

	return &RBACResult{
		ReqOK:           allowed,
		MatchPolicyName: matchPolicyName,
	}, err
}
