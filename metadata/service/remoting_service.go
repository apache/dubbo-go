package service

import (
	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/registry"
)

type RemotingMetadataService interface {
	// PublishMetadata publish the medata info of service from report
	PublishMetadata(service string)
	// GetMetadata get the medata info of service from report
	GetMetadata(instance registry.ServiceInstance) (*common.MetadataInfo, error)
	// PublishServiceDefinition will call remote metadata's StoreProviderMetadata to store url info and service definition
	PublishServiceDefinition(url *common.URL) error
}
