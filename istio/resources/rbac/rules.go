package rbac

import envoyrbacconfigv3 "github.com/envoyproxy/go-control-plane/envoy/config/rbac/v3"

type RuleAction int32

const (
	RuleActionAllow RuleAction = 0
	RuleActionDeny  RuleAction = 1
	RuleActionLog   RuleAction = 2
)

// Enum value maps for RBAC_Action.
var (
	RBAC_Action_name = map[int32]string{
		0: "ALLOW",
		1: "DENY",
		2: "LOG",
	}
	RBAC_Action_value = map[string]int32{
		"ALLOW": 0,
		"DENY":  1,
		"LOG":   2,
	}
)

type Rules struct {
	Action   RuleAction
	Policies map[string]*Policy
}

func (r *Rules) Match(headers map[string]string) (bool, string, error) {
	if r.Action == RuleActionLog {
		return true, "", nil
	} else if r.Action == RuleActionAllow {
		// when engine action is ALLOW, return allowed if matched any policy
		for name, policy := range r.Policies {
			if policy.Match(headers) {
				return true, name, nil
			}
		}
		return false, "", nil
	} else if r.Action == RuleActionDeny {
		// when engine action is DENY, return allowed if not matched any policy
		for name, policy := range r.Policies {
			if policy.Match(headers) {
				return false, name, nil
			}
		}
		return true, "", nil
	}

	return false, "", nil
}

func NewRules(rbac *envoyrbacconfigv3.RBAC) (*Rules, error) {
	rules := &Rules{
		Policies: make(map[string]*Policy, 0),
	}
	switch rbac.Action {
	case envoyrbacconfigv3.RBAC_ALLOW:
		rules.Action = RuleActionAllow
	case envoyrbacconfigv3.RBAC_DENY:
		rules.Action = RuleActionDeny
	case envoyrbacconfigv3.RBAC_LOG:
		rules.Action = RuleActionLog
	}

	for rbacPolicyName, rbacPolicy := range rbac.Policies {
		policy, err := NewPolicy(rbacPolicy)
		if err != nil {
			return nil, err
		}
		rules.Policies[rbacPolicyName] = policy
	}

	return rules, nil
}
