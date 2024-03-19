package xds

import (
	"crypto/tls"
	"crypto/x509"
	"dubbo.apache.org/dubbo-go/v3/common/constant"
	"fmt"

	"dubbo.apache.org/dubbo-go/v3/common"
	"dubbo.apache.org/dubbo-go/v3/common/extension"
	"dubbo.apache.org/dubbo-go/v3/istio"
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
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

func (x *XdsTLSProvider) GetServerWorkLoadTLSConfig(url *common.URL) (*tls.Config, error) {

	cfg := &tls.Config{
		GetCertificate: x.GetWorkloadCertificate,
		ClientAuth:     tls.VerifyClientCertIfGiven, // for test only
		//ClientAuth: tls.RequireAndVerifyClientCert, // for prod
		ClientCAs: x.GetCACertPool(),
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			err := x.VerifyPeerCertByServer(rawCerts, verifiedChains)
			if err != nil {
				logger.Errorf("Could not verify client certificate: %v", err)
			}
			return err
		},
		MinVersion:               tls.VersionTLS12,
		CipherSuites:             tlsprovider.PreferredDefaultCipherSuites(),
		PreferServerCipherSuites: true,
	}

	return cfg, nil
}

func (x *XdsTLSProvider) VerifyPeerCertByServer(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	logger.Infof("[xds tls] server verifiy peer cert")
	if len(rawCerts) == 0 {
		// Peer doesn't present a certificate. Just skip. Other authn methods may be used.
		return nil
	}
	var peerCert *x509.Certificate
	intCertPool := x509.NewCertPool()
	for id, rawCert := range rawCerts {
		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			return err
		}
		if id == 0 {
			peerCert = cert
		} else {
			intCertPool.AddCert(cert)
		}
	}
	if len(peerCert.URIs) != 1 {
		return fmt.Errorf("peer certificate does not contain 1 URI type SAN, detected %d", len(peerCert.URIs))
	}
	spiffe := peerCert.URIs[0].String()
	_, err := resources.ParseIdentity(spiffe)
	if err != nil {
		return err
	}
	secretCache := x.pilotAgent.GetSecretCache()
	hostInboundListener := x.pilotAgent.GetHostInboundListener()
	if hostInboundListener == nil {
		return fmt.Errorf("can not get xds inbound listner info")
	}

	spiffeMatch := hostInboundListener.TransportSocket.SubjectAltNamesMatch
	spiffeValue := hostInboundListener.TransportSocket.SubjectAltNamesValue
	ok := x.matchSpiffeUrl(spiffe, spiffeMatch, spiffeValue)
	if !ok {
		return fmt.Errorf("client spiffe urll %s can not match %s:%s", spiffe, spiffeMatch, spiffeValue)
	}
	rootCertPool, err := secretCache.GetCACertPool()
	if err != nil {
		return fmt.Errorf("no cert pool found ")
	}
	_, err = peerCert.Verify(x509.VerifyOptions{
		Roots:         rootCertPool,
		Intermediates: intCertPool,
	})
	return err
}

func (x *XdsTLSProvider) GetClientWorkLoadTLSConfig(url *common.URL) (*tls.Config, error) {

	verifyMap := make(map[string]string, 0)
	verifyMap["SubjectAltNamesMatch"] = url.GetParam(constant.TLSSubjectAltNamesMatchKey, "")
	verifyMap["SubjectAltNamesValue"] = url.GetParam(constant.TLSSubjectAltNamesValueKey, "")

	cfg := &tls.Config{
		GetCertificate:     x.GetWorkloadCertificate,
		InsecureSkipVerify: false,
		RootCAs:            x.GetCACertPool(),
		VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			certVerifyMap := verifyMap
			err := x.VerifyPeerCertByClient(rawCerts, verifiedChains, certVerifyMap)
			if err != nil {
				logger.Errorf("Could not verify server certificate: %v", err)
			}
			return err
		},
		MinVersion:               tls.VersionTLS12,
		CipherSuites:             tlsprovider.PreferredDefaultCipherSuites(),
		PreferServerCipherSuites: true,
	}

	return cfg, nil

}

func (x *XdsTLSProvider) VerifyPeerCertByClient(rawCerts [][]byte, verifiedChains [][]*x509.Certificate, certVerifyMap map[string]string) error {
	logger.Infof("[xds tls] client verifiy peer cert")
	if len(rawCerts) == 0 {
		// Peer doesn't present a certificate. Just skip. Other authn methods may be used.
		return nil
	}
	var peerCert *x509.Certificate
	intCertPool := x509.NewCertPool()
	for id, rawCert := range rawCerts {
		cert, err := x509.ParseCertificate(rawCert)
		if err != nil {
			return err
		}
		if id == 0 {
			peerCert = cert
		} else {
			intCertPool.AddCert(cert)
		}
	}
	if len(peerCert.URIs) != 1 {
		return fmt.Errorf("peer certificate does not contain 1 URI type SAN, detected %d", len(peerCert.URIs))
	}
	spiffe := peerCert.URIs[0].String()
	_, err := resources.ParseIdentity(spiffe)
	if err != nil {
		return err
	}
	secretCache := x.pilotAgent.GetSecretCache()
	spiffeMatch := certVerifyMap["SubjectAltNamesMatch"]
	spiffeValue := certVerifyMap["SubjectAltNamesValue"]
	ok := x.matchSpiffeUrl(spiffe, spiffeMatch, spiffeValue)
	if !ok {
		return fmt.Errorf("client spiffe urll %s can not match %s:%s", spiffe, spiffeMatch, spiffeValue)
	}
	rootCertPool, err := secretCache.GetCACertPool()
	if err != nil {
		return fmt.Errorf("no cert pool found ")
	}
	_, err = peerCert.Verify(x509.VerifyOptions{
		Roots:         rootCertPool,
		Intermediates: intCertPool,
	})
	return err
}

func (x *XdsTLSProvider) GetWorkloadCertificate(helloInfo *tls.ClientHelloInfo) (*tls.Certificate, error) {
	logger.Infof("[xds tls] get workload certificate")
	secretCache := x.pilotAgent.GetSecretCache()
	tlsCertifcate, err := secretCache.GetWorkloadCertificate(helloInfo)
	if err != nil {
		logger.Errorf("[xds tls] get workload certifcate fail: %v", err)
		return nil, err
	}
	return tlsCertifcate, nil
}

func (x *XdsTLSProvider) GetCACertPool() *x509.CertPool {
	logger.Infof("[xds tls] get ca cert pool")
	secretCache := x.pilotAgent.GetSecretCache()
	certPool, err := secretCache.GetCACertPool()
	if err != nil {
		logger.Errorf("[xds tls] CA cert pool fail: %v", err)
		return nil
	}
	return certPool
}

func (x *XdsTLSProvider) matchSpiffeUrl(spiffe string, match string, value string) bool {
	return resources.MatchSpiffe(spiffe, match, value)
}

func NewXdsTlsProvider() *XdsTLSProvider {
	logger.Infof("[xds tls] init pilot agent")
	pilotAgent, err := istio.GetPilotAgent(istio.PilotAgentTypeServerWorkload)
	if err != nil {
		logger.Errorf("[xds tls] init pilot agent err:%", err)
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
