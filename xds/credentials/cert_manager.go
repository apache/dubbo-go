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

package credentials

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

import (
	log "dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/xds/client/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/xds/credentials/certgenerate"
	"dubbo.apache.org/dubbo-go/v3/xds/credentials/certprovider"
	"dubbo.apache.org/dubbo-go/v3/xds/credentials/certprovider/remote"
	"dubbo.apache.org/dubbo-go/v3/xds/utils/envconfig"
)

//CertManager manage agent or no agent cert
type CertManager interface {
	GetCertificate() ([]tls.Certificate, error)
	GetRootCertificate() (*x509.CertPool, error)
}

// NewCertManager return a manager
func NewCertManager() (CertManager, error) {
	bootstrapPath := os.Getenv(envconfig.XDSBootstrapFileNameEnv)
	if bootstrapPath != "" {
		manager := &AgentCertManager{}
		config, err := bootstrap.NewConfig()
		if err != nil {
			log.Errorf("build bootstrap config error :%s", err.Error())
			return nil, err
		}
		certProvider, err := buildProviderFunc(config.CertProviderConfigs, "default", "rootCA", false, true)

		if err != nil {
			log.Errorf("get cert provider error :%s", err.Error())
			return nil, err
		}
		manager.provider = certProvider
		return manager, nil
	} else {
		manager := &CACertManager{}
		manager.rootPath = RootCertPath
		return manager, nil
	}

}

// AgentCertManager work in istio agent mode
type AgentCertManager struct {
	provider certprovider.Provider
}

//GetRootCertificate return certificate of ca
func (c *AgentCertManager) GetRootCertificate() (*x509.CertPool, error) {
	material, err := c.provider.KeyMaterial(context.Background())
	if err != nil {
		return nil, err
	}
	return material.Roots, nil
}

//GetCertificate return certificate of application
func (c *AgentCertManager) GetCertificate() ([]tls.Certificate, error) {
	material, err := c.provider.KeyMaterial(context.Background())
	if err != nil {
		return nil, err
	}
	return material.Certs, nil
}

func buildProviderFunc(configs map[string]*certprovider.BuildableConfig, instanceName, certName string, wantIdentity, wantRoot bool) (certprovider.Provider, error) {
	cfg, ok := configs[instanceName]
	if !ok {
		return nil, fmt.Errorf("certificate provider instance %q not found in bootstrap file", instanceName)
	}
	provider, err := cfg.Build(certprovider.BuildOptions{
		CertName:     certName,
		WantIdentity: wantIdentity,
		WantRoot:     wantRoot,
	})
	if err != nil {
		return nil, fmt.Errorf("xds: failed to get security plugin instance (%+v): %v", cfg, err)
	}
	return provider, nil
}

// CACertManager work in no agent mode, fetch cert form CA
type CACertManager struct {
	// Certs contains a slice of cert/key pairs used to prove local identity.
	Certs []tls.Certificate
	// Roots contains the set of trusted roots to validate the peer's identity.
	Roots *x509.CertPool

	NoAfter time.Time

	RootNoAfter time.Time

	rootPath string
}

//GetCertificate return certificate of application
func (c *CACertManager) GetCertificate() ([]tls.Certificate, error) {
	//cert expired
	if time.Now().After(c.NoAfter) {
		if err := c.UpdateCert(); err != nil {
			return nil, err
		}
	}
	return c.Certs, nil
}

//GetRootCertificate return certificate of ca
func (c *CACertManager) GetRootCertificate() (*x509.CertPool, error) {
	//root expired
	if time.Now().After(c.RootNoAfter) {
		if err := c.UpdateRoot(); err != nil {
			return nil, err
		}
	}
	return c.Roots, nil
}

func (c *CACertManager) UpdateRoot() error {
	rootFileContents, err := ioutil.ReadFile(c.rootPath)
	if err != nil {
		return err
	}
	trustPool := x509.NewCertPool()
	if !trustPool.AppendCertsFromPEM(rootFileContents) {
		log.Warn("failed to parse root certificate")
	}
	c.Roots = trustPool
	block, _ := pem.Decode(rootFileContents)
	if block == nil {
		return fmt.Errorf("failed to decode certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil
	}
	c.RootNoAfter = cert.NotAfter
	return nil
}

func (c *CACertManager) UpdateCert() error {
	tokenProvider, err := NewSaTokenProvider(IsitoCaServiceAccountPath)
	if err != nil {
		return err
	}

	trustRoot, err := c.GetRootCertificate()
	if err != nil {
		return nil
	}
	citadelClient, err := remote.NewCitadelClient(&remote.Options{
		CAEndpoint:    IstioCAEndpoint,
		TrustedRoots:  trustRoot,
		TokenProvider: tokenProvider,
		CertSigner:    "kubernetes.default.svc",
		ClusterID:     "Kubernetes",
	})
	host := "spiffe://" + "cluster.local" + "/ns/" + "echo-grpc" + "/sa/" + tokenProvider.Token

	//use default
	options := certgenerate.CertOptions{
		Host:       host,
		RSAKeySize: 2048,
		PKCS8Key:   true,
		ECSigAlg:   certgenerate.EcdsaSigAlg,
	}

	// Generate the cert/key, send CSR to CA.
	csrPEM, keyPEM, err := certgenerate.GenCSR(options)
	if err != nil {
		log.Errorf("failed to generate key and certificate for CSR: %v", err)
		return err
	}
	sign, err := citadelClient.CSRSign(csrPEM, int64(200000))

	if err != nil {
		return err
	}

	cert, _, err := c.ParseCert(concatCerts(sign), keyPEM)
	if err != nil {
		return err
	}
	c.Certs = []tls.Certificate{*cert}
	return nil
}

func (c *CACertManager) ParseCert(certByte []byte, keyByte []byte) (*tls.Certificate, time.Time, error) {
	block, _ := pem.Decode(certByte)
	if block == nil {
		return nil, time.Now(), fmt.Errorf("failed to decode certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)

	expired := cert.NotAfter
	log.Infof("cert expired after:" + expired.String())
	c.NoAfter = expired
	pair, err := tls.X509KeyPair(certByte, keyByte)
	if err != nil {
		return nil, time.Now(), fmt.Errorf("failed to parse certificate: %v", err)
	}
	return &pair, expired, nil
}

func concatCerts(certsPEM []string) []byte {
	if len(certsPEM) == 0 {
		return []byte{}
	}
	var certChain bytes.Buffer
	for i, c := range certsPEM {
		certChain.WriteString(c)
		if i < len(certsPEM)-1 && !strings.HasSuffix(c, "\n") {
			certChain.WriteString("\n")
		}
	}
	return certChain.Bytes()
}
