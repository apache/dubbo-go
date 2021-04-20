package remote

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/registry"
)

// RemoteMetadataService for save and get metadata
type RemoteMetadataService interface {
	// PublishMetadata publish the medata info of service from report
	PublishMetadata(service string)
	// GetMetadata get the medata info of service from report
	GetMetadata(instance registry.ServiceInstance) (*common.MetadataInfo, error)
	// PublishServiceDefinition will call remote metadata's StoreProviderMetadata to store url info and service definition
	PublishServiceDefinition(url *common.URL) error
}
