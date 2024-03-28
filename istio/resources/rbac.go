package resources

import (
	"bytes"
	
	rbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	jsonp "github.com/golang/protobuf/jsonpb"
)

// RBACEnvoyFilter definine RBAC filter configuration
type RBACEnvoyFilter struct {
	Name string
	RBAC *rbacv3.RBAC
}

func ParseJsonToRBAC(jsonConf string) (*rbacv3.RBAC, error) {
	rbac := rbacv3.RBAC{}
	un := jsonp.Unmarshaler{
		AllowUnknownFields: true,
	}
	if err := un.Unmarshal(bytes.NewReader([]byte(jsonConf)), &rbac); err != nil {
		return nil, err
	}
	return &rbac, nil
}
