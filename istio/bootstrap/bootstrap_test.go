package bootstrap

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_parseBootstrap(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		wantErr         bool
		wantNodeId      string
		wantSdsGrpcPath string
		wantXdsGrpcPath string
	}{
		{
			name:            "normal case",
			path:            "./testdata/envoy-rev.json",
			wantErr:         false,
			wantNodeId:      "sidecar~10.10.241.72~.~.svc.cluster.local",
			wantSdsGrpcPath: "./var/run/dubbomesh/workload-spiffe-uds/socket",
			wantXdsGrpcPath: "./var/run/dubbomesh/proxy/XDS",
		},

		{
			name:            "bad case",
			path:            "./testdata/envoy-rev-noexist.json",
			wantErr:         true,
			wantNodeId:      "",
			wantSdsGrpcPath: "",
			wantXdsGrpcPath: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseBootstrap(tt.path)
			if tt.wantErr && err == nil {
				t.Errorf("parseBootstrap path %s, err %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if !tt.wantErr && err != nil {
				t.Errorf("parseBootstrap path %s, err %v, wantErr %v", tt.path, err, tt.wantErr)
				return
			}
			if err == nil {
				assert.Equal(t, tt.wantNodeId, got.Node.Id)
				assert.Equal(t, tt.wantXdsGrpcPath, got.XdsGrpcPath)
				assert.Equal(t, tt.wantSdsGrpcPath, got.SdsGrpcPath)
			}

		})
	}
}

func Test_getBootstrapContentTimeout(t *testing.T) {

	tests := []struct {
		name    string
		path    string
		want    string
		wantErr string
	}{
		{
			name:    "normal case",
			path:    "./testdata/envoy-rev.json",
			want:    "sidecar~10.10.241.72~.~.svc.cluster.local",
			wantErr: "",
		},

		{
			name:    "timeout normal case",
			path:    "./var/run/dubbomesh/proxy/noexist.json",
			want:    "",
			wantErr: "timeout",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getBootstrapContentTimeout(tt.path)
			if err != nil {
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("getBootstrapContentTimeout() error = %v, wantErr %v", err, tt.wantErr)
				}
				return
			}
			if !strings.Contains(got, tt.want) {
				t.Errorf("getBootstrapContentTimeout() got = %v, want %v", got, tt.want)
			}
		})
	}
}
