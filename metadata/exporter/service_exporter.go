package exporter

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"dubbo.apache.org/dubbo-go/v3/config"
	"dubbo.apache.org/dubbo-go/v3/metadata"
	"github.com/dubbogo/gost/log/logger"
)

func init() {
	metadata.SetExporterFactory(NewMetadataServiceExporter)
}

// MetadataServiceExporter is the ConfigurableMetadataServiceExporter which implement MetadataServiceExporter interface
type MetadataServiceExporter struct {
	app, metadataType string
	ServiceConfig     *config.ServiceConfig
}

func NewMetadataServiceExporter(app, metadataType string) metadata.MetadataServiceExporter {
	return &MetadataServiceExporter{app: app, metadataType: metadataType}
}

// Export will export the metadataService
func (exporter *MetadataServiceExporter) Export() error {
	exporter.ServiceConfig = config.NewServiceConfigBuilder().
		SetServiceID(constant.SimpleMetadataServiceName).
		SetProtocolIDs(constant.DefaultProtocol).
		AddRCProtocol(constant.DefaultProtocol, config.NewProtocolConfigBuilder().
			SetName(constant.DefaultProtocol).
			Build()).
		SetRegistryIDs("N/A").
		SetInterface(constant.MetadataServiceName).
		SetGroup(exporter.app).
		SetVersion(metadata.GlobalMetadataService.Version()).
		SetProxyFactoryKey(constant.DefaultKey).
		SetMetadataType(exporter.metadataType).
		Build()
	exporter.ServiceConfig.Implement(metadata.GlobalMetadataService)
	err := exporter.ServiceConfig.Export()
	url := exporter.ServiceConfig.GetExportedUrls()[0]
	metadata.GlobalMetadataService.SetMetadataServiceURL(url)
	logger.Infof("[Metadata Service] The MetadataService exports urls : %v ", exporter.ServiceConfig.GetExportedUrls())
	return err
}

// UnExport will unExport the metadataService
func (exporter *MetadataServiceExporter) UnExport() {
	exporter.ServiceConfig.Unexport()
}
