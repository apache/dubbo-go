package resources

import (
	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
)

// JwtAuthnEnovyFilter definine jwt authn filter configuration
type RBACEnvoyFilter struct {
	Name string
	RBAC *rbacv3.RBAC
}
