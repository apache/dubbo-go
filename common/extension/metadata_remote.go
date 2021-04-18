package extension

import (
	"fmt"
)

import (
	perrors "github.com/pkg/errors"
)

import (
	"github.com/apache/dubbo-go/metadata/service/remote"
)

type remoteMetadataServiceCreator func() (remote.RemoteMetadataService, error)

var (
	creator remoteMetadataServiceCreator
)

// SetMetadataRemoteService will store the
func SetMetadataRemoteService(creatorFunc remoteMetadataServiceCreator) {
	creator = creatorFunc
}

// GetRemoteMetadataServiceFactory will create a MetadataService instance
func GetRemoteMetadataService() (remote.RemoteMetadataService, error) {
	if creator != nil {
		return creator()
	}
	return nil, perrors.New(fmt.Sprintf("could not find the metadata service creator for metadataType: remote, please check whether you have imported relative packages, \n" +
		"remote - github.com/apache/dubbo-go/metadata/remote/impl"))
}
