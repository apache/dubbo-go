package utils

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestGetLocalIP(t *testing.T) {
	ip, err := GetLocalIP()
	assert.NoError(t, err)
	t.Log(ip)
}
