package remote

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/registry"
)

type RemoteMetadataService interface {
	PublishMetadata(service string)
	GetMetadata(instance registry.ServiceInstance) (*common.MetadataInfo, error)
	PublishServiceDefinition(url *common.URL) error
}
