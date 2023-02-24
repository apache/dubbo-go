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
	"context"
	"io/ioutil"
)

//provide k8s service account
type saTokenProvider struct {
	Token     string
	tokenPath string
}

// NewSaTokenProvider return a provider
func NewSaTokenProvider(tokenPath string) (*saTokenProvider, error) {
	sa, err := ioutil.ReadFile(tokenPath)
	if err != nil {
		return nil, err
	}
	return &saTokenProvider{
		tokenPath: tokenPath,
		Token:     string(sa),
	}, nil
}

// GetRequestMetadata return meta of authorization
func (s *saTokenProvider) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {

	meta := make(map[string]string)

	meta["authorization"] = "Bearer " + s.Token
	return meta, nil
}

// RequireTransportSecurity always false
func (s *saTokenProvider) RequireTransportSecurity() bool {
	return false
}
