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

package triple_protocol

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

import (
	"github.com/stretchr/testify/require"

	"google.golang.org/protobuf/types/known/emptypb"
)

func TestConfiguredJSONContentTypeAliases(t *testing.T) {
	for _, allowed := range []bool{true, false} {
		codecs := BuiltinSerializationCodecs()
		messages := map[string]Codec{"proto": codecs["proto"]}
		if allowed {
			messages["json"] = codecs["json"]
		}
		_, options := ConfiguredSerializationOptions(codecs["proto"], map[string]Codec{}, messages, false, false)
		handler := NewUnaryHandler("/codec.Alias/Call", func() any { return &emptypb.Empty{} }, func(context.Context, *Request) (*Response, error) { return NewResponse(&emptypb.Empty{}), nil }, options...)
		for _, contentType := range []string{"application/json", "application/json; charset=utf-8", "application/json;Charset=Utf-8"} {
			req := httptest.NewRequest(http.MethodPost, "/codec.Alias/Call", strings.NewReader("{}"))
			req.Header.Set("Content-Type", contentType)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, req)
			expected := http.StatusOK
			if !allowed {
				expected = http.StatusBadRequest
			}
			require.Equal(t, expected, response.Code, "allowed=%t type=%s body=%s", allowed, contentType, response.Body.String())
		}
		req := httptest.NewRequest(http.MethodPost, "/codec.Alias/Call", strings.NewReader("{}"))
		req.Header.Set("Content-Type", "application/json; charset=shift-jis")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, req)
		require.Equal(t, http.StatusUnsupportedMediaType, response.Code)
		for _, name := range []string{codecNameJSON, codecNameJSONCharsetUTF8} {
			session := &tripleServerCodecSession{delegate: &protoJSONCodec{name: name}, runtimeCodecs: map[string]Codec{}, messageCodecs: messages}
			err := session.Unmarshal([]byte("{}"), &emptypb.Empty{})
			if allowed {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, "disabled")
			}
		}
	}
}
