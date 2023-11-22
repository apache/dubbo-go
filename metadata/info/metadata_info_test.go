package info

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common"
)

func TestMetadataInfoAddService(t *testing.T) {
	metadataInfo := &MetadataInfo{
		Services:              make(map[string]*ServiceInfo),
		exportedServiceURLs:   make(map[string][]*common.URL),
		subscribedServiceURLs: make(map[string][]*common.URL),
	}

	url, _ := common.NewURL("dubbo://127.0.0.1:20000?application=foo&category=providers&check=false&dubbo=dubbo-go+v1.5.0&interface=com.foo.Bar&methods=GetPetByID%2CGetPetTypes&organization=Apache&owner=foo&revision=1.0.0&side=provider&version=1.0.0")
	metadataInfo.AddService(url)
	assert.True(t, len(metadataInfo.Services) > 0)
	assert.True(t, len(metadataInfo.exportedServiceURLs) > 0)

	metadataInfo.RemoveService(url)
	assert.True(t, len(metadataInfo.Services) == 0)
	assert.True(t, len(metadataInfo.exportedServiceURLs) == 0)
}
