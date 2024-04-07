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

// Rules define the set of policies to be applied to incoming requests.
// See istio define documents for more details. https://istio.io/latest/docs/reference/config/security/authorization-policy/#Rule
// See istio define rule condition for detials. https://istio.io/latest/docs/reference/config/security/conditions/
// And there is an example rule define as bellow:
/* an example rule
rules:
  policies:
    ns[foo]-policy[httpbin]-rule[0]:
      permissions:
      - Rule:
          AndRules:
            rules:
            - Rule:
                OrRules:
                  rules:
                  - Rule:
                      Header:
                        name: ":method"
                        HeaderMatchSpecifier:
                          StringMatch:
                            MatchPattern:
                              Exact: "GET"
            - Rule:
                OrRules:
                  rules:
                  - Rule:
                      UrlPath:
                        Rule:
                          Path:
                            MatchPattern:
                              Prefix: "/info"
      principals:
      - Identifier:
          AndIds:
            ids:
            - Identifier:
                OrIds:
                  ids:
                  - Identifier:
                      Authenticated:
                        principal_name:
                          MatchPattern:
                            Exact: "spiffe://cluster.local/ns/default/sa/sleep"
            - Identifier:
                OrIds:
                  ids:
                  - Identifier:
                      Metadata:
                        filter: "istio_authn"
                        path:
                        - Segment:
                            Key: "request.auth.claims"
                        - Segment:
                            Key: "iss"
                        value:
                          MatchPattern:
                            ListMatch:
                              MatchPattern:
                                OneOf:
                                  MatchPattern:
                                    StringMatch:
                                      MatchPattern:
                                        Exact: "dubbo.apache.org"
            - Identifier:
                OrIds:
                  ids:
                  - Identifier:
                      Header:
                        name: "User-Agent"
                        HeaderMatchSpecifier:
                          StringMatch:
                            MatchPattern:
                              Prefix: "Mozilla/"
            - Identifier:
                OrIds:
                  ids:
                  - Identifier:
                      DirectRemoteIp:
                        address_prefix: "10.1.2.3"
                        prefix_len:
                          value: 32
                  - Identifier:
                      DirectRemoteIp:
                        address_prefix: "10.2.0.0"
                        prefix_len:
                          value: 16
            - Identifier:
                OrIds:
                  ids:
                  - Identifier:
                      Metadata:
                        filter: "istio_authn"
                        path:
                        - Segment:
                            Key: "request.auth.principal"
                        value:
                          MatchPattern:
                            StringMatch:
                              Exact: "issuer.example.com/subject-admin"
            - Identifier:
                OrIds:
                  ids:
                  - Identifier:
                      Authenticated:
                        principal_name:
                          MatchPattern:
                            Exact: "spiffe://cluster.local/ns/default/sa/productpage"
            - Identifier:
                OrIds:
                  ids:
                  - Identifier:
                      Metadata:
                        filter: "istio_authn"
                        path:
                        - Segment:
                            Key: "request.auth.audiences"
                        value:
                          MatchPattern:
                            StringMatch:
                              Exact: "a.example.com"
                  - Identifier:
                      Metadata:
                        filter: "istio_authn"
                        path:
                        - Segment:
                            Key: "request.auth.audiences"
                        value:
                          MatchPattern:
                            StringMatch:
                              Exact: "b.example.com"
            - Identifier:
                OrIds:
                  ids:
                  - Identifier:
                      Metadata:
                        filter: "istio_authn"
                        path:
                        - Segment:
                            Key: "request.auth.claims"
                        - Segment:
                            Key: "iss"
                        value:
                          MatchPattern:
                            ListMatch:
                              MatchPattern:
                                OneOf:
                                  MatchPattern:
                                    StringMatch:
                                      MatchPattern:
                                        Suffix: "@foo.com"
            - Identifier:
                OrIds:
                  ids:
                  - Identifier:
                      Metadata:
                        filter: "istio_authn"
                        path:
                        - Segment:
                            Key: "request.auth.claims"
                        - Segment:
                            Key: "nestedkey1"
                        - Segment:
                            Key: "nestedkey2"
                        value:
                          MatchPattern:
                            ListMatch:
                              MatchPattern:
                                OneOf:
                                  MatchPattern:
                                    StringMatch:
                                      MatchPattern:
                                        Exact: "some-value1"
                  - Identifier:
                      Metadata:
                        filter: "istio_authn"
                        path:
                        - Segment:
                            Key: "request.auth.claims"
                        - Segment:
                            Key: "nestedkey1"
                        - Segment:
                            Key: "nestedkey2"
                        value:
                          MatchPattern:
                            ListMatch:
                              MatchPattern:
                                OneOf:
                                  MatchPattern:
                                    StringMatch:
                                      MatchPattern:
                                        Exact: "some-value2"
*/

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
	if rbac == nil {
		return nil, nil
	}
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
