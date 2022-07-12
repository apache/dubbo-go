package polaris

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"github.com/polarismesh/polaris-go/api"
	"reflect"
	"sync"
	"testing"
)

func Test_createDeregisterParam(t *testing.T) {
	type args struct {
		url         *common.URL
		serviceName string
	}
	tests := []struct {
		name string
		args args
		want *api.InstanceDeRegisterRequest
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createDeregisterParam(tt.args.url, tt.args.serviceName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createDeregisterParam() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_polarisRegistry_Destroy(t *testing.T) {
	type fields struct {
		url          *common.URL
		provider     api.ProviderAPI
		lock         *sync.RWMutex
		registryUrls map[string]*PolarisHeartbeat
		listenerLock *sync.RWMutex
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "Test_polarisRegistry_Destroy",
			fields: fields{
				url:          nil,
				provider:     nil,
				registryUrls: nil,
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
		})
	}
}
