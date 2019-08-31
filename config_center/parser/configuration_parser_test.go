package parser

import (
	"testing"
)
import (
	"github.com/stretchr/testify/assert"
)

func TestDefaultConfigurationParser_Parser(t *testing.T) {
	parser := &DefaultConfigurationParser{}
	m, err := parser.Parse("dubbo.registry.address=172.0.0.1\ndubbo.registry.name=test")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(m))
	assert.Equal(t, "172.0.0.1", m["dubbo.registry.address"])
}
