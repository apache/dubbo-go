package auth

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	"github.com/apache/dubbo-go/common/extension"
	"github.com/apache/dubbo-go/filter"
	"github.com/apache/dubbo-go/protocol"
)

type DefaultAccesskeyStorage struct {
}

func (storage *DefaultAccesskeyStorage) GetAccessKeyPair(invocation protocol.Invocation, url *common.URL) *filter.AccessKeyPair {
	return &filter.AccessKeyPair{
		AccessKey: url.GetParam(constant.ACCESS_KEY_ID_KEY, ""),
		SecretKey: url.GetParam(constant.SECRET_ACCESS_KEY_KEY, ""),
	}
}

func init() {
	extension.SetAccesskeyStorages(constant.DEFAULT_ACCESS_KEY_STORAGE, GetDefaultAccesskeyStorage)
}

func GetDefaultAccesskeyStorage() filter.AccessKeyStorage {
	return &DefaultAccesskeyStorage{}
}
