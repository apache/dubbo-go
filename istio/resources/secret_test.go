package resources

import (
	"os"
	"testing"

	"crypto/tls"
	"github.com/stretchr/testify/assert"
)

func TestSecretCache_GetCACertPool(t *testing.T) {
	certificateChainBytes, _ := os.ReadFile("./testdata/tls.crt")
	privateKeyBytes, _ := os.ReadFile("./testdata/tls.key")
	rootCertBytes, _ := os.ReadFile("./testdata/ca.crt")
	secretCache := &SecretCache{}
	// Set certificate chain, private key, and root cert bytes in SecretCache
	secretCache.workload = &SecretItem{
		CertificateChain: certificateChainBytes,
		PrivateKey:       privateKeyBytes,
	}
	secretCache.certRoot = rootCertBytes
	_, err1 := secretCache.GetCACertPool()
	assert.NoError(t, err1)
	_, err2 := secretCache.GetClientWorkloadCertificate(&tls.CertificateRequestInfo{})
	assert.NoError(t, err2)
}
