package xds

import (
	"crypto/tls"
	"crypto/x509"

	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/istio"
	tlsprovider "dubbo.apache.org/dubbo-go/v3/tls"
	"github.com/dubbogo/gost/log/logger"
)

const (
	XdsProvider = "xds-provider"
)

var (
	xdsTLSProvider *XdsTLSProvider
)

func init() {
	extension.SetTLSProvider(XdsProvider, GetXdsTLSProvider)
}

type XdsTLSProvider struct {
	pilotAgent *istio.PilotAgent
}

func (x *XdsTLSProvider) GetServerWorkLoadTLSConfig() (*tls.Config, error) {

	cfg := &tls.Config{
		GetCertificate: x.GetWorkloadCertificate,
		ClientAuth:     tls.VerifyClientCertIfGiven, // for test only
		//ClientAuth:     tls.RequireAndVerifyClientCert, // for prod
		ClientCAs: x.GetCACertPool(),
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			err := x.VerifyPeerCert(rawCerts, verifiedChains)
			if err != nil {
				logger.Errorf("Could not verify certificate: %v", err)
			}
			return err
		},
		MinVersion:               tls.VersionTLS12,
		CipherSuites:             tlsprovider.PreferredDefaultCipherSuites(),
		PreferServerCipherSuites: true,
	}

	return cfg, nil
}

func (x *XdsTLSProvider) VerifyPeerCert(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	logger.Infof("[xds tls] verifiy peer cert")
	return nil
}

func (x *XdsTLSProvider) GetClientWorkLoadTLSConfig() (*tls.Config, error) {
	//TODO implement me
	panic("implement me")
}

func (x *XdsTLSProvider) GetWorkloadCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	logger.Infof("[xds tls] get workload certificate")
	return nil, nil
}

func (x *XdsTLSProvider) GetCACertPool() *x509.CertPool {
	logger.Infof("[xds tls] get ca cert pool")
	return nil
}

func NewXdsTlsProvider() *XdsTLSProvider {
	logger.Infof("[xds tls] init pilot agent")
	pilotAgent, err := istio.GetPilotAgent(istio.PilotAgentTypeServerWorkload)
	if err != nil {
		logger.Error("[xds tls] init pilot agent err:%", err)
	}
	return &XdsTLSProvider{
		pilotAgent: pilotAgent,
	}
}

func GetXdsTLSProvider() tlsprovider.TLSProvider {
	if xdsTLSProvider == nil {
		xdsTLSProvider = NewXdsTlsProvider()
	}
	return xdsTLSProvider
}
