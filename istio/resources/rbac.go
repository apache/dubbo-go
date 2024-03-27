package resources

import (
	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
)

// JwtAuthnFilter definine jwt authn filter configuration
type RBACFilter struct {
	Name string
	RBAC *rbacv3.RBAC
}
