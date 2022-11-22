package polaris

import (
	"dubbo.apache.org/dubbo-go/v3/registry"
	"github.com/polarismesh/polaris-go/api"
	"reflect"
	"testing"
)

func Test_convertToDeregisterInstance(t *testing.T) {
	type args struct {
		namespace string
		instance  registry.ServiceInstance
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
			if got := convertToDeregisterInstance(tt.args.namespace, tt.args.instance); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertToDeregisterInstance() = %v, want %v", got, tt.want)
			}
		})
	}
}
