package tls

type TLSProvider interface {
	GetServerWorkLoadTlsConfig() error
	GetClientWorkLoadTlsConfig() error
}
