package mock

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/metadata/mapping"
	"dubbo.apache.org/dubbo-go/v3/metadata/report"
	gxset "github.com/dubbogo/gost/container/set"
	"reflect"
	"testing"
)

func TestMockMetadataFactoryCreateMetadataReport(t *testing.T) {
	type args struct {
		url *common.URL
	}
	tests := []struct {
		name string
		args args
		want report.MetadataReport
	}{
		{
			name: "default",
			args: args{
				url: nil,
			},
			want: &mockMetadataReport{
				appMetadata: make(map[string]*info.MetadataInfo, 0),
				mapping:     make(map[string]*gxset.HashSet, 0),
				listeners:   make(map[string][]mapping.MappingListener, 0),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mo := MockMetadataFactory{}
			if got := mo.CreateMetadataReport(tt.args.url); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateMetadataReport() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMockMetadataReportGetAppMetadata(t *testing.T) {
	type fields struct {
		appMetadata map[string]*info.MetadataInfo
		mapping     map[string]*gxset.HashSet
		listeners   map[string][]mapping.MappingListener
	}
	type args struct {
		application string
		revision    string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *info.MetadataInfo
		wantErr bool
	}{
		{},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockMetadataReport{
				appMetadata: tt.fields.appMetadata,
				mapping:     tt.fields.mapping,
				listeners:   tt.fields.listeners,
			}
			got, err := m.GetAppMetadata(tt.args.application, tt.args.revision)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetAppMetadata() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAppMetadata() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMockMetadataReportGetServiceAppMapping(t *testing.T) {
	type fields struct {
		appMetadata map[string]*info.MetadataInfo
		mapping     map[string]*gxset.HashSet
		listeners   map[string][]mapping.MappingListener
	}
	type args struct {
		interfaceName string
		group         string
		l             mapping.MappingListener
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *gxset.HashSet
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &mockMetadataReport{
				appMetadata: tt.fields.appMetadata,
				mapping:     tt.fields.mapping,
				listeners:   tt.fields.listeners,
			}
			got, err := m.GetServiceAppMapping(tt.args.interfaceName, tt.args.group, tt.args.l)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetServiceAppMapping() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetServiceAppMapping() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMockMetadataReportPublishAppMetadata(t *testing.T) {
	type fields struct {
		appMetadata map[string]*info.MetadataInfo
		mapping     map[string]*gxset.HashSet
		listeners   map[string][]mapping.MappingListener
	}
	type args struct {
		application string
		revision    string
		info        *info.MetadataInfo
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
			m := &mockMetadataReport{
				appMetadata: tt.fields.appMetadata,
				mapping:     tt.fields.mapping,
				listeners:   tt.fields.listeners,
			}
			if err := m.PublishAppMetadata(tt.args.application, tt.args.revision, tt.args.info); (err != nil) != tt.wantErr {
				t.Errorf("PublishAppMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMockMetadataReportRegisterServiceAppMapping(t *testing.T) {
	type fields struct {
		appMetadata map[string]*info.MetadataInfo
		mapping     map[string]*gxset.HashSet
		listeners   map[string][]mapping.MappingListener
	}
	type args struct {
		interfaceName string
		group         string
		application   string
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
			m := &mockMetadataReport{
				appMetadata: tt.fields.appMetadata,
				mapping:     tt.fields.mapping,
				listeners:   tt.fields.listeners,
			}
			if err := m.RegisterServiceAppMapping(tt.args.interfaceName, tt.args.group, tt.args.application); (err != nil) != tt.wantErr {
				t.Errorf("RegisterServiceAppMapping() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMockMetadataReportRemoveServiceAppMappingListener(t *testing.T) {
	type fields struct {
		appMetadata map[string]*info.MetadataInfo
		mapping     map[string]*gxset.HashSet
		listeners   map[string][]mapping.MappingListener
	}
	type args struct {
		interfaceName string
		group         string
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
			m := &mockMetadataReport{
				appMetadata: tt.fields.appMetadata,
				mapping:     tt.fields.mapping,
				listeners:   tt.fields.listeners,
			}
			if err := m.RemoveServiceAppMappingListener(tt.args.interfaceName, tt.args.group); (err != nil) != tt.wantErr {
				t.Errorf("RemoveServiceAppMappingListener() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
