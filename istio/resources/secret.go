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

func (s *SecretCache) GetWorkloadCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
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

func (s *SecretCache) GetCACertPool() (*x509.CertPool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.GetRoot() == nil {
		return nil, fmt.Errorf("can not find root certifcate")
	}
	pool := x509.NewCertPool()
	pool.AppendCertsFromPEM(s.GetRoot())
	return pool, nil
}
