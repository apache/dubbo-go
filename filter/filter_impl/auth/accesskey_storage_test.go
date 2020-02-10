package auth

import (
	"net/url"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
	"github.com/apache/dubbo-go/common/constant"
	invocation2 "github.com/apache/dubbo-go/protocol/invocation"
)

func TestDefaultAccesskeyStorage_GetAccesskeyPair(t *testing.T) {
	url := common.NewURLWithOptions(
		common.WithParams(url.Values{}),
		common.WithParamsValue(constant.SECRET_ACCESS_KEY_KEY, "skey"),
		common.WithParamsValue(constant.ACCESS_KEY_ID_KEY, "akey"))
	invocation := &invocation2.RPCInvocation{}
	storage := GetDefaultAccesskeyStorage()
	accesskeyPair := storage.GetAccessKeyPair(invocation, url)
	assert.Equal(t, "skey", accesskeyPair.SecretKey)
	assert.Equal(t, "akey", accesskeyPair.AccessKey)
}
