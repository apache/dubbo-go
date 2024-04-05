package rbac

type RBACAction int32

const (
	RBACAllow RBACAction = 0
	RBACDeny  RBACAction = 1
	RBACLog   RBACAction = 2
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

type Authorization struct {
	Action   RBACAction
	Policies map[string]*Policy
}
