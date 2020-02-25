package config

import "testing"

import (
	"github.com/stretchr/testify/assert"
)

func TestMetadataReportConfig_ToUrl(t *testing.T) {
	metadataReportConfig := MetadataReportConfig{
		Address:    "mock://127.0.0.1",
		Username:   "test",
		Password:   "test",
		TimeoutStr: "3s",
		Params: map[string]string{
			"k": "v",
		},
	}
	url, error := metadataReportConfig.ToUrl()
	assert.NoError(t, error)
	assert.Equal(t, "mock", url.Protocol)
	assert.Equal(t, "test", url.Username)
	assert.Equal(t, "test", url.Password)
	assert.Equal(t, "v", url.GetParam("k", ""))
	assert.Equal(t, "mock", url.GetParam("metadata", ""))
}
