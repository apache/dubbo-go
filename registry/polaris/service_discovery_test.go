package polaris

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"
	gxset "github.com/dubbogo/gost/container/set"
	gxpage "github.com/dubbogo/gost/hash/page"
	"github.com/polarismesh/polaris-go"
	"github.com/polarismesh/polaris-go/api"
	"reflect"
	"sync"
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

func Test_convertToRegisterInstance(t *testing.T) {
	type args struct {
		namespace string
		instance  registry.ServiceInstance
	}
	tests := []struct {
		name string
		args args
		want *api.InstanceRegisterRequest
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := convertToRegisterInstance(tt.args.namespace, tt.args.instance); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("convertToRegisterInstance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_newPolarisServiceDiscovery(t *testing.T) {
	type args struct {
		url *common.URL
	}
	tests := []struct {
		name    string
		args    args
		want    registry.ServiceDiscovery
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newPolarisServiceDiscovery(tt.args.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("newPolarisServiceDiscovery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newPolarisServiceDiscovery() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_polarisServiceDiscovery_AddListener(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	type args struct {
		listener registry.ServiceInstancesChangedListener
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			if err := polaris.AddListener(tt.args.listener); (err != nil) != tt.wantErr {
				t.Errorf("AddListener() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_polarisServiceDiscovery_Destroy(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			if err := polaris.Destroy(); (err != nil) != tt.wantErr {
				t.Errorf("Destroy() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_polarisServiceDiscovery_GetDefaultPageSize(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			if got := polaris.GetDefaultPageSize(); got != tt.want {
				t.Errorf("GetDefaultPageSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_polarisServiceDiscovery_GetHealthyInstancesByPage(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	type args struct {
		serviceName string
		offset      int
		pageSize    int
		healthy     bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   gxpage.Pager
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			if got := polaris.GetHealthyInstancesByPage(tt.args.serviceName, tt.args.offset, tt.args.pageSize, tt.args.healthy); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetHealthyInstancesByPage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_polarisServiceDiscovery_GetInstances(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	type args struct {
		serviceName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []registry.ServiceInstance
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			if got := polaris.GetInstances(tt.args.serviceName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetInstances() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_polarisServiceDiscovery_GetInstancesByPage(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	type args struct {
		serviceName string
		offset      int
		pageSize    int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   gxpage.Pager
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			if got := polaris.GetInstancesByPage(tt.args.serviceName, tt.args.offset, tt.args.pageSize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetInstancesByPage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_polarisServiceDiscovery_GetRequestInstances(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	type args struct {
		serviceNames  []string
		offset        int
		requestedSize int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   map[string]gxpage.Pager
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			if got := polaris.GetRequestInstances(tt.args.serviceNames, tt.args.offset, tt.args.requestedSize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRequestInstances() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_polarisServiceDiscovery_GetServices(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	tests := []struct {
		name   string
		fields fields
		want   *gxset.HashSet
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			if got := polaris.GetServices(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetServices() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_polarisServiceDiscovery_Register(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	type args struct {
		instance registry.ServiceInstance
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			if err := polaris.Register(tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("Register() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_polarisServiceDiscovery_String(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			if got := polaris.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_polarisServiceDiscovery_Unregister(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	type args struct {
		instance registry.ServiceInstance
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			if err := polaris.Unregister(tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("Unregister() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_polarisServiceDiscovery_Update(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	type args struct {
		instance registry.ServiceInstance
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			if err := polaris.Update(tt.args.instance); (err != nil) != tt.wantErr {
				t.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_polarisServiceDiscovery_createPolarisWatcherIfAbsent(t *testing.T) {
	type fields struct {
		namespace         string
		descriptor        string
		provider          polaris.ProviderAPI
		consumer          polaris.ConsumerAPI
		services          *gxset.HashSet
		instanceLock      *sync.RWMutex
		registryInstances map[string]*PolarisInstanceInfo
		watchers          map[string]*PolarisServiceWatcher
		listenerLock      *sync.RWMutex
	}
	type args struct {
		serviceName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *PolarisServiceWatcher
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			polaris := &polarisServiceDiscovery{
				namespace:         tt.fields.namespace,
				descriptor:        tt.fields.descriptor,
				provider:          tt.fields.provider,
				consumer:          tt.fields.consumer,
				services:          tt.fields.services,
				instanceLock:      tt.fields.instanceLock,
				registryInstances: tt.fields.registryInstances,
				watchers:          tt.fields.watchers,
				listenerLock:      tt.fields.listenerLock,
			}
			got, err := polaris.createPolarisWatcherIfAbsent(tt.args.serviceName)
			if (err != nil) != tt.wantErr {
				t.Errorf("createPolarisWatcherIfAbsent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("createPolarisWatcherIfAbsent() got = %v, want %v", got, tt.want)
			}
		})
	}
}
