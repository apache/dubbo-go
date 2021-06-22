package common

import "dubbo.apache.org/dubbo-go/v3/common/constant"

func IsGeneric(generic string) bool {
	return constant.GENERIC_SERIALIZATION_DEFAULT == generic || constant.GENERIC_SERIALIZATION_JSONRPC == generic
}
