// Copyright 2021-2023 Buf Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package connect_test

import (
	"errors"
	"io"
	"log"
	"net/http"

	"dubbo.apache.org/dubbo-go/v3/protocol/grpc_new/connect"
)

// NewHelloHandler is an example HTTP handler. In a real application, it might
// handle RPCs, requests for HTML, or anything else.
func NewHelloHandler() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		io.WriteString(response, "Hello, world!")
	})
}

// NewAuthenticatedHandler is an example of middleware that works with both RPC
// and non-RPC clients.
func NewAuthenticatedHandler(handler http.Handler) http.Handler {
	errorWriter := connect.NewErrorWriter()
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		// Dummy authentication logic.
		if request.Header.Get("Token") == "super-secret" {
			handler.ServeHTTP(response, request)
			return
		}
		defer request.Body.Close()
		defer io.Copy(io.Discard, request.Body)
		if errorWriter.IsSupported(request) {
			// Send a protocol-appropriate error to RPC clients, so that they receive
			// the right code, message, and any metadata or error details.
			unauthenticated := connect.NewError(connect.CodeUnauthenticated, errors.New("invalid token"))
			errorWriter.Write(response, request, unauthenticated)
		} else {
			// Send an error to non-RPC clients.
			response.WriteHeader(http.StatusUnauthorized)
			io.WriteString(response, "invalid token")
		}
	})
}

func ExampleErrorWriter() {
	mux := http.NewServeMux()
	mux.Handle("/", NewHelloHandler())
	srv := &http.Server{
		Addr:    ":8080",
		Handler: NewAuthenticatedHandler(mux),
	}
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalln(err)
	}
}
