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

package xds

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	log "dubbo.apache.org/dubbo-go/v3/common/logger"
	"dubbo.apache.org/dubbo-go/v3/xds/credentials/certgenerate"
	certprovider2 "dubbo.apache.org/dubbo-go/v3/xds/credentials/certprovider"
	"dubbo.apache.org/dubbo-go/v3/xds/credentials/certprovider/remote"
	"dubbo.apache.org/dubbo-go/v3/xds/utils/envconfig"
	"encoding/pem"
	"fmt"
	v3corepb "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"io/ioutil"
	"os"
	"strings"
)

import (
	xdsCommon "dubbo.apache.org/dubbo-go/v3/remoting/xds/common"
	"dubbo.apache.org/dubbo-go/v3/remoting/xds/mapping"
	"dubbo.apache.org/dubbo-go/v3/xds/client"
	"dubbo.apache.org/dubbo-go/v3/xds/client/bootstrap"
	"dubbo.apache.org/dubbo-go/v3/xds/client/resource/version"
)

// xdsClientFactoryFunction generates new xds client
// when running ut, it's for for ut to replace
var xdsClientFactoryFunction = func(localIP, podName, namespace string, istioAddr xdsCommon.HostAddr) (client.XDSClient, error) {
	// todo fix these ugly magic num
	v3NodeProto := &v3corepb.Node{
		Id:                   "sidecar~" + localIP + "~" + podName + "." + namespace + "~" + namespace + ".svc.cluster.local",
		UserAgentName:        gRPCUserAgentName,
		UserAgentVersionType: &v3corepb.Node_UserAgentVersion{UserAgentVersion: "1.45.0"},
		ClientFeatures:       []string{clientFeatureNoOverprovisioning},
		Metadata:             mapping.GetDubboGoMetadata(""),
	}

	certificates, err := buildCertificate()
	if err != nil {
		return nil, err
	}

	nonNilCredsConfigV2 := &bootstrap.Config{
		XDSServer: &bootstrap.ServerConfig{
			ServerURI:    istioAddr.String(),
			Creds:        grpc.WithTransportCredentials(certificates),
			TransportAPI: version.TransportV3,
			NodeProto:    v3NodeProto,
		},
		ClientDefaultListenerResourceNameTemplate: "%s",
	}

	newClient, err := client.NewWithConfig(nonNilCredsConfigV2)
	if err != nil {
		return nil, err
	}
	return newClient, nil
}

var buildCertificate = func() (credentials.TransportCredentials, error) {

	bootstrapPath := os.Getenv(envconfig.XDSBootstrapFileNameEnv)
	//agent exist,  get cert from file
	if "" != bootstrapPath {
		config, err := bootstrap.NewConfig()
		if err != nil {
			return nil, err
		}
		certProvider, err := buildProviderFunc(config.CertProviderConfigs, "default", "rootCA", false, true)

		material, err := certProvider.KeyMaterial(context.Background())

		if err != nil {
			log.Errorf("failed to get root ca  :%s", err.Error())
			return nil, err
		}

		cred := credentials.NewTLS(&tls.Config{
			RootCAs:      material.Roots,
			Certificates: material.Certs,
			GetClientCertificate: func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				keyMaterial, err := certProvider.KeyMaterial(info.Context())
				if err != nil {
					log.Errorf("failed to get keyMaterial  :%s", err.Error())
					return nil, err
				}
				return &keyMaterial.Certs[0], nil
			},
		})
		return cred, nil

		//agent not exist, get cert from istio citadel
	} else {
		tokenProvider, err := NewSaTokenProvider()
		if err != nil {
			return nil, err
		}
		citadelClient, err := remote.NewCitadelClient(&remote.Options{
			CAEndpoint:    envconfig.IstioCAEndpoint,
			TokenProvider: tokenProvider,
		})

		//use default
		options := certgenerate.CertOptions{
			Host:       "dubbo-go",
			RSAKeySize: 2048,
			PKCS8Key:   true,
			ECSigAlg:   certgenerate.EcdsaSigAlg,
		}

		// Generate the cert/key, send CSR to CA.
		csrPEM, keyPEM, err := certgenerate.GenCSR(options)
		if err != nil {
			log.Errorf("failed to generate key and certificate for CSR: %v", err)
			return nil, err
		}
		sign, err := citadelClient.CSRSign(csrPEM, int64(0))

		if err != nil {
			return nil, err
		}

		cert, err := ParseCert(concatCerts(sign), keyPEM)
		if err != nil {
			return nil, err
		}

		cred := credentials.NewTLS(&tls.Config{
			RootCAs:      nil, //todo fetch rootCA
			Certificates: []tls.Certificate{*cert},
			GetClientCertificate: func(info *tls.CertificateRequestInfo) (*tls.Certificate, error) {
				//todo check cert expired
				return cert, nil
			},
		})
		return cred, nil
	}

}

//provide k8s service account
type saTokenProvider struct {
	token string
}

func NewSaTokenProvider() (*saTokenProvider, error) {
	sa, err := ioutil.ReadFile(envconfig.KubernetesServiceAccountPath)
	if err != nil {
		return nil, err
	}
	return &saTokenProvider{
		token: string(sa),
	}, nil
}

func (s *saTokenProvider) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {

	meta := make(map[string]string)

	meta["authorization"] = "Bearer " + s.token
	return meta, nil
}

func (s *saTokenProvider) RequireTransportSecurity() bool {
	return false
}

func buildProviderFunc(configs map[string]*certprovider2.BuildableConfig, instanceName, certName string, wantIdentity, wantRoot bool) (certprovider2.Provider, error) {
	cfg, ok := configs[instanceName]
	if !ok {
		return nil, fmt.Errorf("certificate provider instance %q not found in bootstrap file", instanceName)
	}
	provider, err := cfg.Build(certprovider2.BuildOptions{
		CertName:     certName,
		WantIdentity: wantIdentity,
		WantRoot:     wantRoot,
	})
	if err != nil {
		// This error is not expected since the bootstrap process parses the
		// config and makes sure that it is acceptable to the plugin. Still, it
		// is possible that the plugin parses the config successfully, but its
		// Build() method errors out.
		return nil, fmt.Errorf("xds: failed to get security plugin instance (%+v): %v", cfg, err)
	}
	return provider, nil
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

func ParseCert(certByte []byte, keyByte []byte) (*tls.Certificate, error) {
	block, _ := pem.Decode(certByte)
	if block == nil {
		return nil, fmt.Errorf("failed to decode certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)

	expired := cert.NotAfter
	log.Infof("cert expired after:" + expired.String())

	pair, err := tls.X509KeyPair(certByte, keyByte)
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %v", err)
	}
	return &pair, nil
}
