package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetLocalIp(t *testing.T) {
	assert.NotNil(t, GetLocalIp())
}
