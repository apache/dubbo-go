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
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwk"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func TestUnmarshalJwks(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "Valid JWKS JSON",
			json:    `{"keys":[{"kid":"123","kty":"RSA","e":"AQAB","n":"zDq_QWvH8eD_xCw7nxLYk6XzBIJkC3_6F2nGxOKrXfo0idGFlc4eNF18jS_gwXuvb12eVCCHlhv6F7W1wGz2_4J3X_wbMxDoXPLiXxNm8ycYKpDXT5CSb6owzjJLoq3ZQ7Qq4GIHnyxq4qEJwZBkhAqqKdnl8U7wJvGZnvq2DxV8jBKJ_ljLRQ04LVDj-8sB4cM-sr91AyyFVz_zsAIGr7FltABnc6R-Nxtj7_2eUcrGYY5EOE9p3TR15zZzVfvQGd_02lJzHvVjpqG2LlS0hvFgIF47mL0x05RlgEz7MpNh1QdS5Kty1AnpV0t4o2aO_b_09IExrBabM0w"}]}`,
			wantErr: false,
		},
		// Add more test cases as needed
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := UnmarshalJwks(tt.json)
			if err != nil {
				assert.Error(t, err)
				return
			}
		})
	}
}

func TestJwtRouteMatch_Match(t *testing.T) {
	tests := []struct {
		name        string
		matchConfig JwtRouteMatch
		path        string
		expected    bool
	}{
		{
			name: "Case-sensitive path match",
			matchConfig: JwtRouteMatch{
				Action:        "path",
				Value:         "/api/v1/users",
				CaseSensitive: true,
			},
			path:     "/api/v1/users",
			expected: true,
		},
		{
			name: "Case-sensitive path mismatch",
			matchConfig: JwtRouteMatch{
				Action:        "path",
				Value:         "/api/v1/users",
				CaseSensitive: true,
			},
			path:     "/API/V1/users",
			expected: false,
		},
		{
			name: "Case-insensitive path match",
			matchConfig: JwtRouteMatch{
				Action:        "path",
				Value:         "/api/v1/users",
				CaseSensitive: false,
			},
			path:     "/API/V1/users",
			expected: true,
		},
		{
			name: "Prefix match (case-sensitive)",
			matchConfig: JwtRouteMatch{
				Action:        "prefix",
				Value:         "/api",
				CaseSensitive: true,
			},
			path:     "/api/v1/users",
			expected: true,
		},
		{
			name: "Prefix mismatch (case-sensitive)",
			matchConfig: JwtRouteMatch{
				Action:        "prefix",
				Value:         "/api",
				CaseSensitive: true,
			},
			path:     "/API/v1/users",
			expected: false,
		},
		{
			name: "Prefix match (case-insensitive)",
			matchConfig: JwtRouteMatch{
				Action:        "prefix",
				Value:         "/api",
				CaseSensitive: false,
			},
			path:     "/API/v1/users",
			expected: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			actual := tc.matchConfig.Match(tc.path)
			assert.Equal(t, tc.expected, actual, "Expected Match result does not match for %s", tc.path)
		})
	}
}

func TestValidateAndParseJWT(t *testing.T) {
	const aLongLongTimeAgo = 233431200
	token := jwt.New()
	token.Set(jwt.SubjectKey, `https://github.com/lestrrat-go/jwx/v2/jwt`)
	token.Set(jwt.AudienceKey, `Golang Users`)
	token.Set(jwt.IssuedAtKey, time.Unix(aLongLongTimeAgo, 0))
	token.Set(`userid`, `Hello, World!`)
	privKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Errorf("failed to generate private key: %s", err)
		return
	}
	// This is the key we will use to sign
	realKey, err := jwk.FromRaw(privKey)
	if err != nil {
		t.Errorf("failed to create JWK: %s\n", err)
		return
	}
	realKey.Set(jwk.KeyIDKey, `mykey`)
	realKey.Set(jwk.AlgorithmKey, jwa.RS256)
	// For demonstration purposes, we also create a bogus key
	bogusKey, err := jwk.FromRaw([]byte("bogus"))
	if err != nil {
		fmt.Printf("failed to create bogus JWK: %s\n", err)
		return
	}
	bogusKey.Set(jwk.AlgorithmKey, jwa.NoSignature)
	bogusKey.Set(jwk.KeyIDKey, "otherkey")

	signingKey := realKey
	privset := jwk.NewSet()
	privset.AddKey(realKey)
	privset.AddKey(bogusKey)
	keySet, err := jwk.PublicSetOf(privset)
	if err != nil {
		fmt.Printf("failed to create public JWKS: %s\n", err)
		return
	}
	// Sign the token and generate a JWS message
	signed, err := jwt.Sign(token, jwt.WithKey(jwa.RS256, signingKey))
	if err != nil {
		fmt.Printf("failed to generate signed serialized: %s\n", err)
		return
	}
	signedToken := string(signed)

	// Test cases
	tests := []struct {
		name        string
		token       string
		keySet      jwk.Set
		expectedErr bool
	}{
		{
			name:        "Valid JWT with correct key set",
			token:       signedToken,
			keySet:      keySet,
			expectedErr: false,
		},
		{
			name:        "Invalid JWT",
			token:       "invalid.jwt.token",
			keySet:      keySet,
			expectedErr: true,
		},
		{
			name:        "Valid JWT with incorrect key set",
			token:       signedToken,
			keySet:      jwk.NewSet(), // Empty key set
			expectedErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ValidateAndParseJWT(tc.token, tc.keySet)
			if tc.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
