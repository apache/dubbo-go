package resources

import (
	"bytes"

	"dubbo.apache.org/dubbo-go/v3/istio/resources/rbac"
	envoyrbacv3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/rbac/v3"
	jsonp "github.com/golang/protobuf/jsonpb"
)

// RBACEnvoyFilter definine RBAC filter configuration
type RBACEnvoyFilter struct {
	Name string
	RBAC *rbac.RBAC
}

func ParseJsonToRBAC(jsonConf string) (*envoyrbacv3.RBAC, error) {
	rbac := envoyrbacv3.RBAC{}
	un := jsonp.Unmarshaler{
		AllowUnknownFields: true,
	}
	if err := un.Unmarshal(bytes.NewReader([]byte(jsonConf)), &rbac); err != nil {
		return nil, err
	}
	return &rbac, nil
}
