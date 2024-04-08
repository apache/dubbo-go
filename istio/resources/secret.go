/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package resources

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"
	"time"
)

const (
	DefaultSecretName = "default"
	RootCASecretName  = "ROOTCA"
)

type SecretCache struct {
	mu       sync.RWMutex
	workload *SecretItem
	certRoot []byte
}

//func InitSecretCacheFromFile() *SecretCache {
//	certRootPath := "/Users/jun/GolandProjects/dubbo/dubbo-mesh/external/dubbo-go/istio/resources"
//	certificateChainBytes, err := os.ReadFile(filepath.Join(certRootPath, "./testdata/tls.crt"))
//	if err != nil {
//		logger.Errorf("failed to read certificate chain file: %v", err)
//		return nil
//	}
//
//	// Example: Load private key bytes from a file
//	privateKeyBytes, err := os.ReadFile(filepath.Join(certRootPath, "./testdata/tls.key"))
//	if err != nil {
//		logger.Errorf("failed to read private key file: %v", err)
//		return nil
//	}
//	// Example: Load root cert bytes from a file
//	rootCertBytes, err := os.ReadFile(filepath.Join(certRootPath, "./testdata/ca.crt"))
//	if err != nil {
//		logger.Errorf("failed to read root cert file: %v", err)
//		return nil
//	}
//	secretCache := &SecretCache{}
//	// Set certificate chain, private key, and root cert bytes in SecretCache
//	secretCache.workload = &SecretItem{
//		CertificateChain: certificateChainBytes,
//		PrivateKey:       privateKeyBytes,
//	}
//	secretCache.certRoot = rootCertBytes
//
//	return secretCache
//}

func NewSecretCache() *SecretCache {
	secretCache := &SecretCache{}
	return secretCache
}

// SecretItem is the cached item in in-memory secret store.
type SecretItem struct {
	CertificateChain []byte
	PrivateKey       []byte

	RootCert []byte

	// ResourceName passed from envoy SDS discovery request.
	// "ROOTCA" for root cert request, "default" for key/cert request.
	ResourceName string

	CreatedTime time.Time

	ExpireTime time.Time
}

// GetRoot returns cached root cert and cert expiration time. This method is thread safe.
func (s *SecretCache) GetRoot() (rootCert []byte) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.certRoot
}

// SetRoot sets root cert into cache. This method is thread safe.
func (s *SecretCache) SetRoot(rootCert []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.certRoot = rootCert
}

func (s *SecretCache) GetWorkload() *SecretItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.workload == nil {
		return nil
	}
	return s.workload
}

func (s *SecretCache) SetWorkload(value *SecretItem) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.workload = value
}

func (s *SecretCache) GetServerWorkloadCertificate(helloInfo *tls.ClientHelloInfo) (*tls.Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getWorkloadCertificate()
}

func (s *SecretCache) getWorkloadCertificate() (*tls.Certificate, error) {
	if s.workload == nil {
		return nil, fmt.Errorf("workload certificate is missing")
	}
	// decode certificate pem block
	var certs []*x509.Certificate
	certificateChain := s.workload.CertificateChain

	for len(certificateChain) > 0 {
		var certBlock *pem.Block
		certBlock, certificateChain = pem.Decode(certificateChain)
		if certBlock == nil {
			break
		}
		if certBlock.Type == "CERTIFICATE" {
			cert, err := x509.ParseCertificate(certBlock.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse certificate from PEM: %w", err)
			}
			certs = append(certs, cert)
		}
	}
	if len(certs) == 0 {
		return nil, fmt.Errorf("no certificates found in PEM certificate chain")
	}

	// decode private key pem block
	var privKey interface{}
	var err error
	privBlock, _ := pem.Decode(s.workload.PrivateKey)
	if privBlock == nil {
		return nil, fmt.Errorf("no private key found in PEM data")
	}

	switch privBlock.Type {
	case "RSA PRIVATE KEY":
		privKey, err = x509.ParsePKCS1PrivateKey(privBlock.Bytes)
	case "EC PRIVATE KEY":
		privKey, err = x509.ParseECPrivateKey(privBlock.Bytes)
	case "PRIVATE KEY":
		// PKCS8 formate
		privKey, err = x509.ParsePKCS8PrivateKey(privBlock.Bytes)
	default:
		return nil, fmt.Errorf("unknown private key type in PEM: %q", privBlock.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key from PEM: %w", err)
	}

	// here can only use first certificate in certificate chain
	tlsCert := tls.Certificate{
		Certificate: [][]byte{certs[0].Raw},
		PrivateKey:  privKey,
	}
	return &tlsCert, nil
}

func (s *SecretCache) GetClientWorkloadCertificate(requestInfo *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.getWorkloadCertificate()
}

func (s *SecretCache) GetCACertPool() (*x509.CertPool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.certRoot) == 0 {
		return nil, fmt.Errorf("root cert is empty")
	}
	certPool := x509.NewCertPool()
	pemBlock, _ := pem.Decode(s.certRoot)
	if pemBlock == nil {
		return nil, fmt.Errorf("failed to decode PEM block from root cert")
	}
	if pemBlock.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("expected PEM block of type CERTIFICATE, got %s", pemBlock.Type)
	}
	if !certPool.AppendCertsFromPEM(s.certRoot) {
		return nil, fmt.Errorf("failed to append root cert to cert pool")
	}
	return certPool, nil
}

func (s *SecretCache) GetCertificateChainPEM() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.workload.CertificateChain) == 0 {
		return "", fmt.Errorf("certificate chain is empty")
	}
	return string(s.workload.CertificateChain), nil
}

func (s *SecretCache) GetPrivateKeyPEM() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.workload.PrivateKey) == 0 {
		return "", fmt.Errorf("private key is empty")
	}
	return string(s.workload.PrivateKey), nil
}

func (s *SecretCache) GetRootCertPEM() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if len(s.certRoot) == 0 {
		return "", fmt.Errorf("root cert is empty")
	}
	return string(s.certRoot), nil
}
