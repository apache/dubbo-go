package exporter

import (
	"github.com/dubbogo/gost/log/logger"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/metadata"
)

func init() {
	metadata.SetExporterFactory(NewMetadataServiceExporter)
}

// MetadataServiceExporter is the ConfigurableMetadataServiceExporter which implement MetadataServiceExporter interface
type MetadataServiceExporter struct {
	app, metadataType string
	ServiceConfig     *config.ServiceConfig
	service           metadata.MetadataService
}

func NewMetadataServiceExporter(app, metadataType string, service metadata.MetadataService) metadata.ServiceExporter {
	return &MetadataServiceExporter{app: app, metadataType: metadataType, service: service}
}

// Export will export the metadataService
func (exporter *MetadataServiceExporter) Export() error {
	version, _ := exporter.service.Version()
	exporter.ServiceConfig = config.NewServiceConfigBuilder().
		SetServiceID(constant.SimpleMetadataServiceName).
		SetProtocolIDs(constant.DefaultProtocol).
		AddRCProtocol(constant.DefaultProtocol, config.NewProtocolConfigBuilder().
			SetName(constant.DefaultProtocol).
			Build()).
		SetRegistryIDs("N/A").
		SetInterface(constant.MetadataServiceName).
		SetGroup(exporter.app).
		SetVersion(version).
		SetProxyFactoryKey(constant.DefaultKey).
		SetMetadataType(exporter.metadataType).
		Build()
	exporter.ServiceConfig.Implement(exporter.service)
	err := exporter.ServiceConfig.Export()
	if err != nil {
		return err
	}
	url := exporter.ServiceConfig.GetExportedUrls()[0]
	exporter.service.SetMetadataServiceURL(url)
	logger.Infof("[Metadata Service] The MetadataService exports urls : %v ", exporter.ServiceConfig.GetExportedUrls())
	return nil
}

// UnExport will unExport the metadataService
func (exporter *MetadataServiceExporter) UnExport() {
	exporter.ServiceConfig.Unexport()
}
