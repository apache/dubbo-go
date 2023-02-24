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

package config

import (
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
)

// TLSConfig tls config
type TLSConfig struct {
	CACertFile    string `yaml:"ca-cert-file" json:"ca-cert-file" property:"ca-cert-file"`
	TLSCertFile   string `yaml:"tls-cert-file" json:"tls-cert-file" property:"tls-cert-file"`
	TLSKeyFile    string `yaml:"tls-key-file" json:"tls-key-file" property:"tls-key-file"`
	TLSServerName string `yaml:"tls-server-name" json:"tls-server-name" property:"tls-server-name"`
}

// GetServerTlsConfig build server tls config from TLSConfig
func GetServerTlsConfig(opt *TLSConfig) (*tls.Config, error) {
	//no TLS
	if opt.TLSCertFile == "" && opt.TLSKeyFile == "" {
		return nil, nil
	}
	var ca *x509.CertPool
	cfg := &tls.Config{}
	//need mTLS
	if opt.CACertFile != "" {
		ca = x509.NewCertPool()
		caBytes, err := ioutil.ReadFile(opt.CACertFile)
		if err != nil {
			return nil, err
		}
		if ok := ca.AppendCertsFromPEM(caBytes); !ok {
			return nil, err
		}
		cfg.ClientAuth = tls.RequireAndVerifyClientCert
		cfg.ClientCAs = ca
	}
	cert, err := tls.LoadX509KeyPair(opt.TLSCertFile, opt.TLSKeyFile)
	if err != nil {
		return nil, err
	}
	cfg.Certificates = []tls.Certificate{cert}
	cfg.ServerName = opt.TLSServerName

	return cfg, nil
}

// GetClientTlsConfig build client tls config from TLSConfig
func GetClientTlsConfig(opt *TLSConfig) (*tls.Config, error) {
	//no TLS
	if opt.CACertFile == "" {
		return nil, nil
	}
	cfg := &tls.Config{
		ServerName: opt.TLSServerName,
	}
	ca := x509.NewCertPool()
	caBytes, err := ioutil.ReadFile(opt.CACertFile)
	if err != nil {
		return nil, err
	}
	if ok := ca.AppendCertsFromPEM(caBytes); !ok {
		return nil, err
	}
	cfg.RootCAs = ca
	//need mTls
	if opt.TLSCertFile != "" {
		var cert tls.Certificate
		cert, err = tls.LoadX509KeyPair(opt.TLSCertFile, opt.TLSKeyFile)
		if err != nil {
			return nil, err
		}
		cfg.Certificates = []tls.Certificate{cert}
	}
	return cfg, err
}
