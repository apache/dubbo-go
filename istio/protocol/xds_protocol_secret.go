package protocol

import (
	"dubbo.apache.org/dubbo-go/v3/istio/resources"
	"dubbo.apache.org/dubbo-go/v3/istio/utils"
	"github.com/dubbogo/gost/log/logger"
	tls "github.com/envoyproxy/go-control-plane/envoy/extensions/transport_sockets/tls/v3"
	"time"
)

type SecretProtocol struct {
	secretCache *resources.SecretCache
}

func NewSecretProtocol(secretCache *resources.SecretCache) (*SecretProtocol, error) {
	secretProtocol := &SecretProtocol{
		secretCache: secretCache,
	}
	return secretProtocol, nil
}

func (s *SecretProtocol) ProcessSecret(secret *tls.Secret) error {
	logger.Infof("[secret protocol] parse envoy tls secret:%s", utils.ConvertJsonString(secret))
	if secret.GetName() == resources.DefaultSecretName {
		if secret.GetTlsCertificate() != nil {
			certificateChain := secret.GetTlsCertificate().GetCertificateChain().GetInlineBytes()
			privateKey := secret.GetTlsCertificate().GetPrivateKey().GetInlineBytes()
			item := &resources.SecretItem{
				CertificateChain: certificateChain,
				PrivateKey:       privateKey,
				CreatedTime:      time.Now(),
				ResourceName:     secret.GetName(),
				// TODO parse ROOTCA and expire time
			}
			s.secretCache.SetWorkload(item)
		}
	}

	if secret.GetName() == resources.RootCASecretName {
		if secret.GetValidationContext() != nil {
			rootCA := secret.GetValidationContext().GetTrustedCa().GetInlineBytes()
			s.secretCache.SetRoot(rootCA)
		}
	}

	return nil
}
