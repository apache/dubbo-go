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

package tls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"os"
)

import (
	"dubbo.apache.org/dubbo-go/v3/global"
)

// GetServerTlsConfig build server tls config from TLSConfig
func GetServerTlsConfig(tlsConf *global.TLSConfig) (*tls.Config, error) {
	//no TLS
	if tlsConf.TLSCertFile == "" && tlsConf.TLSKeyFile == "" {
		return nil, nil
	}
	var ca *x509.CertPool
	cfg := &tls.Config{}
	//need mTLS
	if tlsConf.CACertFile != "" {
		ca = x509.NewCertPool()
		caBytes, err := os.ReadFile(tlsConf.CACertFile)
		if err != nil {
			return nil, err
		}
		if ok := ca.AppendCertsFromPEM(caBytes); !ok {
			return nil, errors.New("failed to parse root certificate")
		}
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
		cfg.ClientCAs = ca
	}
	cert, err := tls.LoadX509KeyPair(tlsConf.TLSCertFile, tlsConf.TLSKeyFile)
	if err != nil {
		return nil, err
	}
	cfg.Certificates = []tls.Certificate{cert}
	cfg.ServerName = tlsConf.TLSServerName

	return cfg, nil
}

// GetClientTlsConfig build client tls config from TLSConfig
func GetClientTlsConfig(tlsConf *global.TLSConfig) (*tls.Config, error) {
	//no TLS
	if tlsConf.CACertFile == "" {
		return nil, nil
	}
	cfg := &tls.Config{
		ServerName: tlsConf.TLSServerName,
	}
	ca := x509.NewCertPool()
	caBytes, err := os.ReadFile(tlsConf.CACertFile)
	if err != nil {
		return nil, err
	}
	if ok := ca.AppendCertsFromPEM(caBytes); !ok {
		return nil, errors.New("failed to parse root certificate")
	}
	cfg.RootCAs = ca
	//need mTls
	if tlsConf.TLSCertFile != "" {
		var cert tls.Certificate
		cert, err = tls.LoadX509KeyPair(tlsConf.TLSCertFile, tlsConf.TLSKeyFile)
		if err != nil {
			return nil, err
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, err
}
