package rbac

import envoyrbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"

type RBAC struct {
	Rules       *Rules
	ShadowRules *Rules
}

func NewRBAC(envoyRBAC *envoyrbacv3.RBAC) (*RBAC, error) {
	rules, err := NewRules(envoyRBAC.Rules)
	if err != nil {
		return nil, err
	}
	shadowRules, err2 := NewRules(envoyRBAC.ShadowRules)
	if err2 != nil {
		return nil, err2
	}

	rbac := &RBAC{
		Rules:       rules,
		ShadowRules: shadowRules,
	}
	return rbac, nil
}
