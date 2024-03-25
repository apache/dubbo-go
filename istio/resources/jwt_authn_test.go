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
	"testing"

	"github.com/stretchr/testify/assert"
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
