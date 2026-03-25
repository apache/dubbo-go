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
	"net/http"
	"net/http/httptest"
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

const (
	methodRouteMuxGetUserPath      = "/Service/GetUser"
	methodRouteMuxGetUserLowerPath = "/Service/getUser"
	methodRouteMuxNotFoundBody     = "404 page not found\n"
)

func TestMethodRouteMux(t *testing.T) {
	tests := []struct {
		name           string
		register       func(*methodRouteMux)
		requestPath    string
		wantStatusCode int
		wantBody       string
	}{
		{
			name: "exact match",
			register: func(m *methodRouteMux) {
				m.Handle(methodRouteMuxGetUserPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("exact"))
				}))
			},
			requestPath:    methodRouteMuxGetUserPath,
			wantStatusCode: http.StatusOK,
			wantBody:       "exact",
		},
		{
			name: "lowercase first rune fallback",
			register: func(m *methodRouteMux) {
				m.Handle(methodRouteMuxGetUserPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("fallback"))
				}))
			},
			requestPath:    methodRouteMuxGetUserLowerPath,
			wantStatusCode: http.StatusOK,
			wantBody:       "fallback",
		},
		{
			name: "uppercase first rune fallback",
			register: func(m *methodRouteMux) {
				m.Handle(methodRouteMuxGetUserLowerPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("fallback"))
				}))
			},
			requestPath:    methodRouteMuxGetUserPath,
			wantStatusCode: http.StatusOK,
			wantBody:       "fallback",
		},
		{
			name: "full lowercase fallback",
			register: func(m *methodRouteMux) {
				m.Handle(methodRouteMuxGetUserPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("fallback"))
				}))
			},
			requestPath:    "/service/getuser",
			wantStatusCode: http.StatusNotFound,
			wantBody:       methodRouteMuxNotFoundBody,
		},
		{
			name: "not found",
			register: func(m *methodRouteMux) {
				m.Handle(methodRouteMuxGetUserPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("unexpected"))
				}))
			},
			requestPath:    "/Service/Delete",
			wantStatusCode: http.StatusNotFound,
			wantBody:       methodRouteMuxNotFoundBody,
		},
		{
			name: "methods differing beyond first rune remain distinct",
			register: func(m *methodRouteMux) {
				m.Handle("/S/GetUser", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("first"))
				}))
				m.Handle("/S/Getuser", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("second"))
				}))
			},
			requestPath:    "/S/getuser",
			wantStatusCode: http.StatusOK,
			wantBody:       "second",
		},
		{
			name: "service path remains case sensitive",
			register: func(m *methodRouteMux) {
				m.Handle(methodRouteMuxGetUserPath, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					_, _ = w.Write([]byte("unexpected"))
				}))
			},
			requestPath:    "/service/getUser",
			wantStatusCode: http.StatusNotFound,
			wantBody:       methodRouteMuxNotFoundBody,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mux := newMethodRouteMux()
			tt.register(mux)

			req := httptest.NewRequest(http.MethodPost, tt.requestPath, nil)
			resp := httptest.NewRecorder()
			mux.ServeHTTP(resp, req)

			assert.Equal(t, tt.wantStatusCode, resp.Code)
			assert.Equal(t, tt.wantBody, resp.Body.String())
		})
	}
}

func TestNormalizeMethodRouteKeyEdgeCases(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{
			name: "path without slash remains unchanged",
			path: "GetUser",
			want: "GetUser",
		},
		{
			name: "path ending with slash remains unchanged",
			path: "/Service/",
			want: "/Service/",
		},
		{
			name: "root path remains unchanged",
			path: "/",
			want: "/",
		},
		{
			name: "normal path lowercases first rune of method only",
			path: methodRouteMuxGetUserPath,
			want: methodRouteMuxGetUserLowerPath,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, normalizeMethodRouteKey(tt.path))
		})
	}
}

func TestLowerFirstRuneEdgeCases(t *testing.T) {
	assert.Empty(t, lowerFirstRune(""))
	assert.Equal(t, "abc", lowerFirstRune("Abc"))
	assert.Equal(t, "äbc", lowerFirstRune("Äbc"))
}

func TestMethodRouteMuxFallbackCollisionFirstRegistrationWins(t *testing.T) {
	mux := newMethodRouteMux()
	firstPattern := methodRouteMuxGetUserPath
	secondPattern := methodRouteMuxGetUserLowerPath

	mux.Handle(firstPattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("first"))
	}))
	mux.Handle(secondPattern, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("second"))
	}))

	lowerKey := normalizeMethodRouteKey(firstPattern)
	mux.mu.RLock()
	entry, ok := mux.lower[lowerKey]
	mux.mu.RUnlock()

	if !assert.True(t, ok) {
		return
	}
	assert.Equal(t, firstPattern, entry.pattern)

	req := httptest.NewRequest(http.MethodPost, firstPattern, nil)
	resp := httptest.NewRecorder()
	entry.handler.ServeHTTP(resp, req)
	assert.Equal(t, "first", resp.Body.String())
}
