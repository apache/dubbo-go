package config

type SslConfig struct {
	ServerKeyCertChainPath        string
	serverPrivateKeyPath          string
	serverKeyPassword             string
	serverTrustCertCollectionPath string
}
