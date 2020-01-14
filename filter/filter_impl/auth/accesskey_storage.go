package auth

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
)

const (
	ACCESS_KEY_ID_KEY = "accessKeyId"

	SECRET_ACCESS_KEY_KEY = "secretAccessKey"
)

type DefaultAccesskeyStorage struct {
}

func (storage *DefaultAccesskeyStorage) GetAccesskeyPair(invocation protocol.Invocation, url *common.URL) *filter.AccessKeyPair {
	return &filter.AccessKeyPair{
		AccessKey: url.GetParam(ACCESS_KEY_ID_KEY, ""),
		SecretKey: url.GetParam(SECRET_ACCESS_KEY_KEY, ""),
	}
}

func init() {
	extension.SetAccesskeyStorages(constant.DEFAULT_ACCESS_KEY_STORAGE, GetDefaultAccesskeyStorage)
}

func GetDefaultAccesskeyStorage() filter.AccesskeyStorage {
	return &DefaultAccesskeyStorage{}
}
