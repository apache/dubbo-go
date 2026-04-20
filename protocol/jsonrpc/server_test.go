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

package jsonrpc

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

import (
	"github.com/stretchr/testify/require"
)

// sendHTTPRequest writes an HTTP request to conn and returns the parsed response.
// A read deadline is set to avoid hanging when the server may not respond
// (e.g. valid content type but no registered service).
func sendHTTPRequest(t *testing.T, conn net.Conn, contentType string) (*http.Response, error) {
	t.Helper()

	err := conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	require.NoError(t, err)

	req := "POST /test HTTP/1.1\r\n" +
		"Host: localhost\r\n" +
		"Content-Type: " + contentType + "\r\n" +
		"Content-Length: 0\r\n" +
		"\r\n"
	_, err = conn.Write([]byte(req))
	require.NoError(t, err)

	resp, err := http.ReadResponse(bufio.NewReader(conn), nil)
	return resp, err
}

func TestHandlePkg_ContentType(t *testing.T) {
	tests := []struct {
		name            string
		contentType     string
		wantUnsupported bool
	}{
		{
			name:            "unsupported content type",
			contentType:     "text/html",
			wantUnsupported: true,
		},
		{
			name:            "malformed content type",
			contentType:     ";;;invalid",
			wantUnsupported: true,
		},
		{
			name:            "json with charset",
			contentType:     "application/json; charset=utf-8",
			wantUnsupported: false,
		},
		{
			name:            "json-rpc with charset",
			contentType:     "application/json-rpc; charset=utf-8",
			wantUnsupported: false,
		},
		{
			name:            "plain json",
			contentType:     "application/json",
			wantUnsupported: false,
		},
		{
			name:            "plain json-rpc",
			contentType:     "application/json-rpc",
			wantUnsupported: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverConn, clientConn := net.Pipe()
			defer func() {
				require.NoError(t, clientConn.Close())
			}()
			defer func() {
				require.NoError(t, serverConn.Close())
			}()

			s := NewServer()
			go s.handlePkg(serverConn)

			resp, err := sendHTTPRequest(t, clientConn, tt.contentType)
			if err != nil {
				t.Fatalf("failed to read response: %v", err)
			}
			defer func() {
				require.NoError(t, resp.Body.Close())
			}()

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			if tt.wantUnsupported {
				require.Equal(t, 500, resp.StatusCode)
				require.Contains(t, string(body), "unsupported content type",
					"response body should contain 'unsupported content type'")
			} else {
				require.NotContains(t, string(body), "unsupported content type",
					"%s should be accepted as valid content type", tt.contentType)
			}
		})
	}
}
