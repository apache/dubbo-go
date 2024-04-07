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
	"encoding/json"
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
	now := time.Now()
	exp := now.Add(10 * 365 * 24 * time.Hour)

	token := jwt.New()
	token.Set(jwt.SubjectKey, "spiffe://cluster.local/ns/dubbo/sa/dubboclient")
	token.Set(jwt.IssuerKey, "dubbo.apache.org")
	token.Set(jwt.AudienceKey, "dev")
	token.Set(jwt.IssuedAtKey, now)
	token.Set(jwt.ExpirationKey, exp)

	token2 := jwt.New()
	token2.Set(jwt.SubjectKey, "spiffe://cluster.local/ns/dubbo/sa/httpbin")
	token2.Set(jwt.IssuerKey, "dubbo.apache.org")
	token2.Set(jwt.AudienceKey, "test")
	token2.Set(jwt.IssuedAtKey, now)
	token2.Set(jwt.ExpirationKey, exp)

	token3 := jwt.New()
	token3.Set(jwt.SubjectKey, "spiffe://cluster.local/ns/dubbo/sa/marketing")
	token3.Set(jwt.IssuerKey, "dubbo.apache.org")
	token3.Set(jwt.AudienceKey, `marketing`)
	token3.Set(jwt.IssuedAtKey, now)
	token3.Set(jwt.ExpirationKey, exp)

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
	realKey.Set(jwk.KeyIDKey, `key1`)
	realKey.Set(jwk.AlgorithmKey, jwa.RS256)

	// For demonstration purposes, we also create a bogus key
	bogusKey, err := jwk.FromRaw([]byte("bogus"))
	if err != nil {
		fmt.Printf("failed to create bogus JWK: %s\n", err)
		return
	}
	bogusKey.Set(jwk.AlgorithmKey, jwa.NoSignature)
	bogusKey.Set(jwk.KeyIDKey, "key2")

	signingKey := realKey

	privset := jwk.NewSet()
	privset.AddKey(realKey)
	//privset.AddKey(bogusKey)
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

	signed2, err := jwt.Sign(token2, jwt.WithKey(jwa.RS256, signingKey))
	if err != nil {
		fmt.Printf("failed to generate signed serialized: %s\n", err)
		return
	}
	signedToken2 := string(signed2)

	signed3, err := jwt.Sign(token3, jwt.WithKey(jwa.RS256, signingKey))
	if err != nil {
		fmt.Printf("failed to generate signed serialized: %s\n", err)
		return
	}
	signedToken3 := string(signed3)

	fmt.Printf("token1: %s\n", signedToken)
	fmt.Printf("token2: %s\n", signedToken2)
	fmt.Printf("token3: %s\n", signedToken3)
	jsonbuf, err := json.Marshal(keySet)
	fmt.Printf("keyset: %s\n", string(jsonbuf))

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

func TestJwtClaims_FlattenMap(t *testing.T) {
	// Example JWT Claims instance
	jc := &JwtClaims{
		Issuer:     "issuer",
		Subject:    "subject",
		Expiration: time.Date(2023, time.January, 1, 0, 0, 0, 0, time.UTC),
		Audience:   []string{"audience1", "audience2"},
		IssuedAt:   time.Date(2022, time.December, 1, 0, 0, 0, 0, time.UTC),
		JWTID:      "unique-jti",
		NotBefore:  time.Date(2022, time.November, 1, 0, 0, 0, 0, time.UTC),
		PrivateClaims: map[string]interface{}{
			"nested": map[string]interface{}{
				"key1": "value1",
				"key2": "value2",
				"key3": true,
				"key4": 100,
				"key5": map[string]interface{}{
					"key51": "hello",
					"key52": true,
				},
			},
			"simple_key": "simple_value",
		},
	}

	tests := []struct {
		name string
		jc   *JwtClaims
		want map[string]string
	}{
		{
			name: "Basic FlattenMap Test",
			jc:   jc,
			want: map[string]string{
				":request.auth.claims.iss":         "issuer",
				":request.auth.claims.sub":         "subject",
				":request.auth.claims.aud":         "audience1,audience2",
				":request.auth.claims.jti":         "unique-jti",
				":request.auth.principal":          "issuer/subject",
				":request.auth.audiences":          "audience1,audience2",
				":request.auth.claims.nested.key1": "value1",
				":request.auth.claims.nested.key2": "value2",
				":request.auth.claims.nested.key3": "true",
				":request.auth.claims.nested.key4": "100",
				":request.auth.claims.simple_key":  "simple_value",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.jc.FlattenMap()
			if len(got) != len(tt.want) {
				t.Errorf("FlattenMap() length mismatch; got %d, want %d", len(got), len(tt.want))
			}

			for k, v := range tt.want {
				if gotV, ok := got[k]; !ok || gotV != v {
					t.Errorf("FlattenMap()[%q] = %v, want %v", k, gotV, v)
				}
			}
		})
	}
}
