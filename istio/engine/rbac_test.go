package engine

import (
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"fmt"
	"os"
	"testing"

	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"dubbo.apache.org/dubbo-go/v3/istio/resources/rbac"
	"github.com/stretchr/testify/assert"
)

func TestRBACFilterEngine_Filter(t *testing.T) {

	tests := []struct {
		file    string
		name    string
		headers map[string]string
		want    *RBACResult
		wantErr bool
	}{
		// deny all
		{
			name: "deny all",
			file: "./testdata/deny-all.json",
			headers: map[string]string{
				"x-request-id": "123456",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "deny-all",
			},
			wantErr: false,
		},

		// principal tests

		{
			name: "principal metadata deny default namespace",
			file: "./testdata/principal-metadata.json",
			headers: map[string]string{
				"x-request-id":      "123456",
				":source.principal": "spiffe://cluster.local/ns/default/httpbin",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "metadata-match",
			},
			wantErr: false,
		},

		{
			name: "principal metadata allow foo namespace",
			file: "./testdata/principal-metadata.json",
			headers: map[string]string{
				"x-request-id":      "123456",
				":source.principal": "spiffe://cluster.local/ns/foo/httpbin",
			},
			want: &RBACResult{
				ReqOK:           true,
				MatchPolicyName: "",
			},
			wantErr: false,
		},

		{
			name: "path deny",
			file: "./testdata/permission-path.json",
			headers: map[string]string{
				"x-request-id": "123456",
				":path":        "/deny",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "path-match",
			},
			wantErr: false,
		},

		{
			name: "path allow",
			file: "./testdata/permission-path.json",
			headers: map[string]string{
				"x-request-id": "123456",
				":path":        "/hello",
			},
			want: &RBACResult{
				ReqOK:           true,
				MatchPolicyName: "",
			},
			wantErr: false,
		},
		{
			name: "principal header value regex-match deny",
			file: "./testdata/principal-headers-value.json",
			headers: map[string]string{
				"x-request-id": "123456",
				":path":        "/deny/me/ok",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "header-regex-match",
			},
			wantErr: false,
		},

		{
			name: "principal header value prefixMatch deny",
			file: "./testdata/principal-headers-value.json",
			headers: map[string]string{
				"x-request-id": "123456",
				":path":        "/control-api/hello",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "header-regex-match",
			},
			wantErr: false,
		},

		{
			name: "principal header value suffixMatch deny",
			file: "./testdata/principal-headers-value.json",
			headers: map[string]string{
				"x-request-id": "123456",
				":path":        "/api/a.html",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "header-regex-match",
			},
			wantErr: false,
		},

		{
			name: "principal header value rangeMatch deny",
			file: "./testdata/principal-headers-value.json",
			headers: map[string]string{
				"x-request-id": "123456",
				"x-timeout":    "101",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "header-regex-match",
			},
			wantErr: false,
		},

		{
			name: "principal and",
			file: "./testdata/principal-and.json",
			headers: map[string]string{
				"x-request-id": "123456",
			},
			want: &RBACResult{
				ReqOK:           true,
				MatchPolicyName: "",
			},
			wantErr: false,
		},

		{
			name: "principal or",
			file: "./testdata/principal-or.json",
			headers: map[string]string{
				"x-request-id": "123456",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "or-match",
			},
			wantErr: false,
		},

		{
			name: "principal not",
			file: "./testdata/principal-not.json",
			headers: map[string]string{
				"x-request-id": "123456",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "",
			},
			wantErr: false,
		},
		{
			name: "principal headers present",
			file: "./testdata/principal-headers-present.json",
			headers: map[string]string{
				"x-request-id":    "123456",
				"x-custom-header": "xx",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "header-present-match",
			},
			wantErr: false,
		},

		// permission tests

		{
			name: "permission and",
			file: "./testdata/permission-and.json",
			headers: map[string]string{
				"x-request-id": "123456",
			},
			want: &RBACResult{
				ReqOK:           true,
				MatchPolicyName: "",
			},
			wantErr: false,
		},

		{
			name: "permission header present",
			file: "./testdata/permission-headers-present.json",
			headers: map[string]string{
				"x-request-id":    "123456",
				"x-custom-header": "xx",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "header-present-match",
			},
			wantErr: false,
		},

		{
			name: "permission header value default",
			file: "./testdata/permission-headers-value.json",
			headers: map[string]string{
				"x-request-id": "123456",
			},
			want: &RBACResult{
				ReqOK:           true,
				MatchPolicyName: "",
			},
			wantErr: false,
		},

		{
			name: "permission header value authority match",
			file: "./testdata/permission-headers-value.json",
			headers: map[string]string{
				"x-request-id": "123456",
				":authority":   "example.org",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "header-regex-match",
			},
			wantErr: false,
		},

		{
			name: "permission header value path prefix match",
			file: "./testdata/permission-headers-value.json",
			headers: map[string]string{
				"x-request-id": "123456",
				":path":        "/control-api/hello",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "header-regex-match",
			},
			wantErr: false,
		},

		{
			name: "permission header value path method match",
			file: "./testdata/permission-headers-value.json",
			headers: map[string]string{
				"x-request-id": "123456",
				":method":      "HEAD",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "header-regex-match",
			},
			wantErr: false,
		},

		{
			name: "permission not",
			file: "./testdata/permission-not.json",
			headers: map[string]string{
				"x-request-id": "123456",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "",
			},
			wantErr: false,
		},

		{
			name: "permission or",
			file: "./testdata/permission-or.json",
			headers: map[string]string{
				"x-request-id": "123456",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "or-match",
			},
			wantErr: false,
		},

		{
			name: "permission and",
			file: "./testdata/permission-and.json",
			headers: map[string]string{
				"x-request-id": "123456",
			},
			want: &RBACResult{
				ReqOK:           true,
				MatchPolicyName: "",
			},
			wantErr: false,
		},
		{
			name: "permission path",
			file: "./testdata/permission-path.json",
			headers: map[string]string{
				"x-request-id": "123456",
				":path":        "/deny",
			},
			want: &RBACResult{
				ReqOK:           false,
				MatchPolicyName: "path-match",
			},
			wantErr: false,
		},

		{
			name: "principal meta data auth",
			file: "./testdata/principal-metadata-auth.json",
			headers: map[string]string{
				"x-request-id":             "123456",
				":request.auth.claims.aud": "dev",
				":method":                  "POST",
				":source.principal":        "spiffe://cluster.local/ns/dubbo/sa/dubboclient",
			},
			want: &RBACResult{
				ReqOK:           true,
				MatchPolicyName: "ns[foo]-policy[httpbin]-rule[0]",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			json, _ := os.ReadFile(tt.file)
			envoyRBAC, err := resources.ParseJsonToRBAC(string(json))
			if err != nil {
				t.Errorf("ParseJsonToRBAC error %v", err)
			}
			rbac, err := rbac.NewRBAC(envoyRBAC)
			fmt.Printf("rbac :%s", utils.ConvertJsonString(rbac))
			if err != nil {
				t.Errorf("rbac.NewRBAC error %v", err)
			}
			r := &RBACFilterEngine{
				RBAC: rbac,
			}
			result, err := r.Filter(tt.headers)
			if err != nil {
				t.Errorf("Filter error %v", err)
				return
			}
			assert.Equalf(t, tt.want, result, "Filter(%v)", tt.headers)
		})
	}
}
