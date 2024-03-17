package tls

type TlsProvider interface {
	GetServerWorkLoadTlsConfig() error
	GetClientWorkLoadTlsConfig() error
}
