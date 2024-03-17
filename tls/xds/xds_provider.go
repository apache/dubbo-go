package xds

import (
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/tls"
)

const (
	XdsProvider = "xds-provider"
)

var (
	xdsTlsProvider *XdsTlsProvider
)

func init() {
	extension.SetTlsProvider(XdsProvider, GetXdsTlsProvider)
}

type XdsTlsProvider struct {
}

func (x XdsTlsProvider) GetServerWorkLoadTlsConfig() error {
	//TODO implement me
	panic("implement me")
}

func (x XdsTlsProvider) GetClientWorkLoadTlsConfig() error {
	//TODO implement me
	panic("implement me")
}

func NewXdsTlsProvider() *XdsTlsProvider {
	return &XdsTlsProvider{}
}

func GetXdsTlsProvider() tls.TlsProvider {
	if xdsTlsProvider == nil {
		xdsTlsProvider = NewXdsTlsProvider()
	}
	return xdsTlsProvider
}
