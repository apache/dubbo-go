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
	"errors"
	"fmt"
	"github.com/dubbogo/gost/log/logger"
	"os"
	"path/filepath"
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

func InitSecretCacheFromFile() *SecretCache {
	certRootPath := "/Users/jun/GolandProjects/dubbo/dubbo-mesh/external/dubbo-go/istio/resources"
	certificateChainBytes, err := os.ReadFile(filepath.Join(certRootPath, "./testdata/tls.crt"))
	if err != nil {
		logger.Errorf("failed to read certificate chain file: %v", err)
		return nil
	}

	// Example: Load private key bytes from a file
	privateKeyBytes, err := os.ReadFile(filepath.Join(certRootPath, "./testdata/tls.key"))
	if err != nil {
		logger.Errorf("failed to read private key file: %v", err)
		return nil
	}
	// Example: Load root cert bytes from a file
	rootCertBytes, err := os.ReadFile(filepath.Join(certRootPath, "./testdata/ca.crt"))
	if err != nil {
		logger.Errorf("failed to read root cert file: %v", err)
		return nil
	}
	secretCache := &SecretCache{}
	// Set certificate chain, private key, and root cert bytes in SecretCache
	secretCache.workload = &SecretItem{
		CertificateChain: certificateChainBytes,
		PrivateKey:       privateKeyBytes,
	}
	secretCache.certRoot = rootCertBytes

	return secretCache
}

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
	if s.workload == nil {
		return nil, fmt.Errorf("can not find workload certifcate")
	}
	certificate, err := tls.X509KeyPair(s.workload.CertificateChain, s.workload.PrivateKey)
	if err != nil {
		return nil, err
	}
	return &certificate, nil
}

func (s *SecretCache) GetClientWorkloadCertificate(requestInfo *tls.CertificateRequestInfo) (*tls.Certificate, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.workload == nil {
		return nil, fmt.Errorf("can not find workload certifcate")
	}
	// get certificate chain and private key bytes
	certPEMBlock := s.workload.CertificateChain
	keyPEMBlock := s.workload.PrivateKey
	// decode pem block
	var certificates []*x509.Certificate
	for len(certPEMBlock) > 0 {
		var block *pem.Block
		block, certPEMBlock = pem.Decode(certPEMBlock)
		if block == nil {
			return nil, errors.New("can not decode pem block")
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, errors.New("can not parse certificate")
		}
		certificates = append(certificates, certificate)
	}
	// decode private key pem block
	keyBlock, _ := pem.Decode(keyPEMBlock)
	if keyBlock == nil {
		return nil, errors.New("can not decode private key pem block")
	}

	// parse private key
	privateKey, err := x509.ParsePKCS1PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, errors.New("can not parse private key")
	}
	// create tls.Certificate
	certificate := &tls.Certificate{
		Certificate: [][]byte{certPEMBlock},
		PrivateKey:  privateKey,
		Leaf:        certificates[0], // first certificate of chain
	}
	return certificate, nil

}

func (s *SecretCache) GetCACertPool() (*x509.CertPool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.GetRoot() == nil {
		return nil, fmt.Errorf("can not find root certifcate")
	}
	pool := x509.NewCertPool()
	block, rest := pem.Decode(s.GetRoot())
	var blockBytes []byte

	// Loop while there are no block are found
	for block != nil {
		blockBytes = append(blockBytes, block.Bytes...)
		block, rest = pem.Decode(rest)
	}

	rootCAs, err := x509.ParseCertificates(blockBytes)
	if err != nil {
		return nil, fmt.Errorf("parse certificate from rootPEM got error: %v", err)
	}

	for _, ca := range rootCAs {
		pool.AddCert(ca)
	}
	return pool, nil

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
