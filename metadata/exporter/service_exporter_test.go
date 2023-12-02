package exporter

import (
	"reflect"
	"testing"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	_ "dubbo.apache.org/dubbo-go/v3/protocol/dubbo"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
)

func TestMetadataServiceExporter_Export(t *testing.T) {
	type fields struct {
		app           string
		metadataType  string
		ServiceConfig *config.ServiceConfig
		service       metadata.MetadataService
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
			exporter := &MetadataServiceExporter{
				app:           tt.fields.app,
				metadataType:  tt.fields.metadataType,
				ServiceConfig: tt.fields.ServiceConfig,
				service:       tt.fields.service,
			}
			if err := exporter.Export(); (err != nil) != tt.wantErr {
				t.Errorf("Export() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMetadataServiceExporter_UnExport(t *testing.T) {
	type fields struct {
		app           string
		metadataType  string
		ServiceConfig *config.ServiceConfig
		service       metadata.MetadataService
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			exporter := &MetadataServiceExporter{
				app:           tt.fields.app,
				metadataType:  tt.fields.metadataType,
				ServiceConfig: tt.fields.ServiceConfig,
				service:       tt.fields.service,
			}
			exporter.UnExport()
		})
	}
}

func TestNewMetadataServiceExporter(t *testing.T) {
	type args struct {
		app          string
		metadataType string
		service      metadata.MetadataService
	}
	tests := []struct {
		name string
		args args
		want metadata.ServiceExporter
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMetadataServiceExporter(tt.args.app, tt.args.metadataType, tt.args.service); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMetadataServiceExporter() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExporter(t *testing.T) {
	err := NewMetadataServiceExporter("test", "local", &Service{}).Export()
	if err != nil {
		panic(err)
	}
	select {}
}

type Service struct {
}

func (s Service) GetExportedURLs(serviceInterface string, group string, version string, protocol string) ([]*common.URL, error) {
	//TODO implement me
	panic("implement me")
}

func (s Service) GetExportedServiceURLs() ([]*common.URL, error) {
	//TODO implement me
	panic("implement me")
}

func (s Service) GetSubscribedURLs() ([]*common.URL, error) {
	//TODO implement me
	panic("implement me")
}

func (s Service) Version() (string, error) {
	return "1.0.0", nil
}

func (s Service) GetMetadataInfo(revision string) (*info.MetadataInfo, error) {
	meta := info.NewMetadataInfo()
	meta.App = "test"
	meta.Revision = "1"
	//meta.Services = make(map[string]*info.ServiceInfo)
	//meta.Services["1"] = info.NewServiceInfo("test", "group", "1.0.0", "dubbo", "", make(map[string]string))
	return meta, nil
}

func (s Service) GetMetadataServiceURL() (*common.URL, error) {
	//TODO implement me
	panic("implement me")
}

func (s Service) SetMetadataServiceURL(url *common.URL) {

}
