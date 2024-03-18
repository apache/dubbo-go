package extension

import "dubbo.apache.org/dubbo-go/v3/tls"

var tlsProviders = make(map[string]func() tls.TLSProvider)

// SetTLSProvider sets the TlsProvider extension with @name
func SetTLSProvider(name string, v func() tls.TLSProvider) {
	tlsProviders[name] = v
}

// GetTLSProvider finds the TlsProvider extension with @name
func GetTLSProvider(name string) tls.TLSProvider {
	if tlsProviders[name] == nil {
		panic("tls provider for " + name + " is not existing, make sure you have import the package.")
	}
	return tlsProviders[name]()
}
