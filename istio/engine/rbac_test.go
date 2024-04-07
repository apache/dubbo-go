package engine

import (
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"dubbo.apache.org/dubbo-go/v3/istio/resources/rbac"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestRBACFilterEngine_Filter(t *testing.T) {

	tests := []struct {
		file    string
		name    string
		headers map[string]string
		want    *RBACResult
		wantErr bool
	}{
		//{
		//	name: "deny all",
		//	file: "./testdata/deny-all.json",
		//	headers: map[string]string{
		//		"x-request-id": "123456",
		//	},
		//	want: &RBACResult{
		//		ReqOK:           false,
		//		MatchPolicyName: "deny-all",
		//	},
		//	wantErr: false,
		//},

		{
			name: "meta deny default namespace",
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
			name: "meta allow foo namespace",
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			json, _ := os.ReadFile(tt.file)
			envoyRBAC, err := resources.ParseJsonToRBAC(string(json))
			if err != nil {
				t.Errorf("ParseJsonToRBAC error %v", err)
			}
			rbac, err := rbac.NewRBAC(envoyRBAC)
			//fmt.Printf("rbac :%s", utils.ConvertJsonString(rbac))
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
