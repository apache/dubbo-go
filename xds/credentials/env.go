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
	"os"
)

const (
	DefaultKubernetesServiceAccountPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	DefaultIsitoCaServiceAccountPath    = "/var/run/secrets/tokens/istio-token"
	DefaultRootCertPath                 = "/var/run/secrets/istio/root-cert.pem"
	DefaultIstioCAEndpoint              = "istiod.istio-system.svc:15012"
	DefaultCertSigner                   = "kubernetes.default.svc"
	DefaultClusterID                    = "Kubernetes"
)

var (
	PodNamespace       = f(os.Getenv("POD_NAMESPACE"), "default")
	CertTTL            = f(os.Getenv("CERT_TTL"), "31536000")
	RootCertPath       = f(os.Getenv("ROOT_CERT_PATH"), DefaultRootCertPath)
	IstioCAEndpoint    = f(os.Getenv("ISTIO_CA_ENDPOINT"), DefaultIstioCAEndpoint)
	ServiceAccountPath = f(os.Getenv("SERVICE_ACCOUNT_PATH"), DefaultIsitoCaServiceAccountPath)
	certSigner         = f(os.Getenv("CERT_SIGNER"), DefaultCertSigner)
	clusterID          = f(os.Getenv("CLUSTER_ID"), DefaultClusterID)
	URIPrefix          = f(os.Getenv("URI_PREFIX"), "spiffe://")
	Domain             = f(os.Getenv("DOMAIN"), "cluster.local")
	ServiceAccountName = f(os.Getenv("SERVICE_ACCOUNT_NAME"), "default")
)

var f = func(a, b string) string {
	if "" == a {
		return b
	}
	return a
}
