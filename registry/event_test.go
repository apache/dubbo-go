package registry

import (
	"github.com/apache/dubbo-go/common"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKey(t *testing.T) {
	u1, _ := common.NewURL("dubbo://127.0.0.1:20000/com.ikurento.user.UserProvider?interface=com.ikurento.user.UserProvider&group=&version=2.0")
	se := ServiceEvent{
		Service: u1,
	}
	assert.Equal(t, se.Key(), "dubbo://:@127.0.0.1:20000/?interface=com.ikurento.user.UserProvider&group=&version=2.0&timestamp=")

	se2 := ServiceEvent{
		Service: u1,
		KeyFunc: defineKey,
	}
	assert.Equal(t, se2.Key(), "Hello Key")
}

func defineKey(url *common.URL) string {
	return "Hello Key"
}
