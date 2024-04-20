package metadata

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/metadata/info"
	"dubbo.apache.org/dubbo-go/v3/protocol"
	_ "dubbo.apache.org/dubbo-go/v3/proxy/proxy_factory"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

var (
	url, _ = common.NewURL("dubbo://127.0.0.1:20000?application=foo&category=providers&check=false&dubbo=dubbo-go+v1.5.0&interface=com.foo.Bar&methods=GetPetByID%2CGetPetTypes&organization=Apache&owner=foo&revision=1.0.0&side=provider&version=1.0.0")
)

func newMetadataMap() map[string]*info.MetadataInfo {
	metadataInfo := info.NewAppMetadataInfo("dubbo-app")
	metadataInfo.Revision = "1"
	metadataInfo.AddService(url)
	metadataInfo.AddSubscribeURL(url)
	registryMetadataInfo["default"] = metadataInfo
	return map[string]*info.MetadataInfo{
		"default": metadataInfo,
	}
}

func TestDefaultMetadataServiceGetExportedServiceURLs(t *testing.T) {
	mts := &DefaultMetadataService{
		metadataMap: newMetadataMap(),
	}
	got, err := mts.GetExportedServiceURLs()
	assert.Nil(t, err)
	assert.True(t, len(got) == 1)
	assert.Equal(t, url, got[0])
}

func TestDefaultMetadataServiceGetExportedURLs(t *testing.T) {
	type args struct {
		serviceInterface string
		group            string
		version          string
		protocol         string
	}
	tests := []struct {
		name string
		args args
		want []*common.URL
	}{
		{
			name: "all exact",
			args: args{
				serviceInterface: url.Interface(),
				group:            url.Group(),
				version:          url.Version(),
				protocol:         url.Protocol,
			},
			want: []*common.URL{url},
		},
		{
			name: "interface *",
			args: args{
				serviceInterface: "*",
				group:            url.Group(),
				version:          url.Version(),
				protocol:         url.Protocol,
			},
			want: []*common.URL{url},
		},
		{
			name: "group *",
			args: args{
				serviceInterface: url.Interface(),
				group:            "*",
				version:          url.Version(),
				protocol:         url.Protocol,
			},
			want: []*common.URL{url},
		},
		{
			name: "version *",
			args: args{
				serviceInterface: url.Interface(),
				group:            url.Group(),
				version:          "*",
				protocol:         url.Protocol,
			},
			want: []*common.URL{url},
		},
		{
			name: "protocol *",
			args: args{
				serviceInterface: url.Interface(),
				group:            url.Group(),
				version:          url.Version(),
				protocol:         "*",
			},
			want: []*common.URL{url},
		},
		{
			name: "all *",
			args: args{
				serviceInterface: "*",
				group:            "*",
				version:          "*",
				protocol:         "*",
			},
			want: []*common.URL{url},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mts := &DefaultMetadataService{
				metadataMap: newMetadataMap(),
			}
			got, err := mts.GetExportedURLs(tt.args.serviceInterface, tt.args.group, tt.args.version, tt.args.protocol)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "GetExportedURLs(%v, %v, %v, %v)", tt.args.serviceInterface, tt.args.group, tt.args.version, tt.args.protocol)
		})
	}
}

func TestDefaultMetadataServiceGetMetadataInfo(t *testing.T) {
	type args struct {
		revision string
	}
	tests := []struct {
		name string
		args args
		want *info.MetadataInfo
	}{
		{
			name: "normal",
			args: args{
				revision: "1",
			},
			want: newMetadataMap()["default"],
		},
		{
			name: "empty revision",
			args: args{
				revision: "",
			},
			want: nil,
		},
		{
			name: "revision not match",
			args: args{
				revision: "2",
			},
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mts := &DefaultMetadataService{
				metadataMap: newMetadataMap(),
			}
			got, err := mts.GetMetadataInfo(tt.args.revision)
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "GetMetadataInfo(%v)", tt.args.revision)
		})
	}
}

func TestDefaultMetadataServiceGetMetadataServiceURL(t *testing.T) {
	type fields struct {
		metadataUrl *common.URL
	}
	tests := []struct {
		name   string
		fields fields
		want   *common.URL
	}{
		{
			name: "normal",
			fields: fields{
				metadataUrl: url,
			},
			want: url,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mts := &DefaultMetadataService{
				metadataUrl: tt.fields.metadataUrl,
			}
			got, err := mts.GetMetadataServiceURL()
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "GetMetadataServiceURL()")
		})
	}
}

func TestDefaultMetadataServiceGetSubscribedURLs(t *testing.T) {
	tests := []struct {
		name string
		want []*common.URL
	}{
		{
			name: "normal",
			want: []*common.URL{url},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mts := &DefaultMetadataService{
				metadataMap: newMetadataMap(),
			}
			got, err := mts.GetSubscribedURLs()
			assert.Nil(t, err)
			assert.Equalf(t, tt.want, got, "GetSubscribedURLs()")
		})
	}
}

func TestDefaultMetadataServiceMethodMapper(t *testing.T) {
	tests := []struct {
		name string
		want map[string]string
	}{
		{
			name: "normal",
			want: map[string]string{
				"GetExportedURLs": "getExportedURLs",
				"GetMetadataInfo": "getMetadataInfo",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mts := &DefaultMetadataService{
				metadataMap: newMetadataMap(),
			}
			assert.Equalf(t, tt.want, mts.MethodMapper(), "MethodMapper()")
		})
	}
}

func TestDefaultMetadataServiceSetMetadataServiceURL(t *testing.T) {
	type args struct {
		url *common.URL
	}
	tests := []struct {
		name string
		args args
		want *common.URL
	}{
		{
			name: "normal",
			args: args{
				url: url,
			},
			want: url,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mts := &DefaultMetadataService{
				metadataMap: map[string]*info.MetadataInfo{},
			}
			mts.SetMetadataServiceURL(tt.args.url)
			assert.Equal(t, tt.want, mts.metadataUrl)
		})
	}
}

func TestDefaultMetadataServiceVersion(t *testing.T) {
	mts := &DefaultMetadataService{}
	got, err := mts.Version()
	assert.Nil(t, err)
	assert.Equal(t, version, got)
}

func Test_randomPort(t *testing.T) {
	port := randomPort()
	assert.True(t, port != "")
}

func Test_serviceExporterExport(t *testing.T) {
	mockExporter := new(mockExporter)
	defer mockExporter.AssertExpectations(t)
	mockProtocol := new(mockProtocol)
	defer mockProtocol.AssertExpectations(t)
	extension.SetProtocol("dubbo", func() protocol.Protocol {
		return mockProtocol
	})
	t.Run("normal", func(t *testing.T) {
		port := randomPort()
		p, err := strconv.Atoi(port)
		assert.Nil(t, err)
		opts := &Options{
			appName:      "dubbo-app",
			metadataType: constant.RemoteMetadataStorageType,
			port:         p,
		}
		mockProtocol.On("Export").Return(mockExporter).Once()
		mockExporter.On("UnExport").Once()
		e := &serviceExporter{
			opts:    opts,
			service: &DefaultMetadataService{},
		}
		err = e.Export()
		assert.Nil(t, err)
		e.UnExport()
	})
	// first t.Run has called commom.ServiceMap.Register ,second will fail
	t.Run("get methods error", func(t *testing.T) {
		port := randomPort()
		p, err := strconv.Atoi(port)
		assert.Nil(t, err)
		opts := &Options{
			appName:      "dubbo-app",
			metadataType: constant.RemoteMetadataStorageType,
			port:         p,
		}
		e := &serviceExporter{
			opts:    opts,
			service: &DefaultMetadataService{},
		}
		err = e.Export()
		assert.NotNil(t, err)
	})
	t.Run("port == 0", func(t *testing.T) {
		opts := &Options{
			appName:      "dubbo-app",
			metadataType: constant.RemoteMetadataStorageType,
			port:         0,
		}
		// UnRegister first otherwise will fail
		err := common.ServiceMap.UnRegister(constant.MetadataServiceName, constant.DefaultProtocol,
			common.ServiceKey(constant.MetadataServiceName, opts.appName, version))
		assert.Nil(t, err)
		mockProtocol.On("Export").Return(mockExporter).Once()
		mockExporter.On("UnExport").Once()
		e := &serviceExporter{
			opts:    opts,
			service: &DefaultMetadataService{},
		}
		err = e.Export()
		assert.Nil(t, err)
		e.UnExport()
	})
}

func Test_serviceExporterUnExport(t *testing.T) {
	mockExporter := new(mockExporter)
	defer mockExporter.AssertExpectations(t)
	serviceExporter := &serviceExporter{protocolExporter: mockExporter}
	mockExporter.On("UnExport").Once()
	serviceExporter.UnExport()
}
