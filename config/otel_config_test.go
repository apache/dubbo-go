package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewOtelConfigBuilder(t *testing.T) {
	config := NewOtelConfigBuilder().Build()
	assert.NotNil(t, config)
	assert.NotNil(t, config.TraceConfig)

	err := config.Init()
	assert.NoError(t, err)

	tpc := config.TraceConfig.toTraceProviderConfig()
	assert.NotNil(t, tpc)
}
