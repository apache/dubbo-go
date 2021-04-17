package extension

import (
	"fmt"
	"github.com/apache/dubbo-go/metadata/remote"
	perrors "github.com/pkg/errors"
)

var (
	creator func() (remote.RemoteMetadataService, error)
)

// SetMetadataRemoteService will store the
func SetMetadataRemoteService(creatorFunc func() (remote.RemoteMetadataService, error)) {
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
