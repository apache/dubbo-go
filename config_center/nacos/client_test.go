package nacos

import (
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"github.com/apache/dubbo-go/common"
)

func Test_newNacosClient(t *testing.T) {
	server := mockCommonNacosServer()
	nacosURL := strings.ReplaceAll(server.URL, "http", "registry")
	registryUrl, _ := common.NewURL(nacosURL)
	c := &nacosDynamicConfiguration{
		url:  &registryUrl,
		done: make(chan struct{}),
	}
	err := ValidateNacosClient(c, WithNacosName(nacosClientName))
	assert.NoError(t, err)
	c.wg.Add(1)
	go HandleClientRestart(c)
	c.client.Close()
	<-c.client.Done()
	c.Destroy()
}
