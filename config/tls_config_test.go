package config

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

import (
	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func TestNewTLSConfigBuilder(t *testing.T) {
	config := NewTLSConfigBuilder().
		SetCACertFile("ca_cert_file").
		SetTLSKeyFile("tls_key_file").
		SetTLSServerName("tls_server_name").
		SetTLSCertFile("tls_cert_file").
		Build()
	assert.Equal(t, config.CACertFile, "ca_cert_file")
	assert.Equal(t, config.TLSCertFile, "tls_cert_file")
	assert.Equal(t, config.TLSServerName, "tls_server_name")
	assert.Equal(t, config.TLSKeyFile, "tls_key_file")
	assert.Equal(t, config.Prefix(), constant.TLSConfigPrefix)

}
