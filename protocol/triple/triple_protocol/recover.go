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

package triple_protocol

import (
	"context"
	"net/http"
)

// recoverHandlerInterceptor lets handlers trap panics, perform side effects
// (like emitting logs or metrics), and present a friendlier error message to
// clients.
//
// This interceptor uses a somewhat unusual strategy to recover from panics.
// The standard recovery idiom:
//
//	if r := recover(); r != nil { ... }
//
// isn't robust in the face of user error, because it doesn't handle
// panic(nil). This occasionally happens by mistake, and it's a beast to debug
// without a more robust idiom. See https://github.com/golang/go/issues/25448
// for details.
type recoverHandlerInterceptor struct {
	Interceptor

	handle func(context.Context, Spec, http.Header, interface{}) error
}

func (i *recoverHandlerInterceptor) WrapUnary(next UnaryFunc) UnaryFunc {
	return func(ctx context.Context, req AnyRequest, resp AnyResponse) (retErr error) {
		if req.Spec().IsClient {
			return next(ctx, req, resp)
		}
		panicked := true
		defer func() {
			if panicked {
				r := recover()
				// net/http checks for ErrAbortHandler with ==, so we should too.
				if r == http.ErrAbortHandler { //nolint:errorlint,goerr113
					panic(r) //nolint:forbidigo
				}
				retErr = i.handle(ctx, req.Spec(), req.Header(), r)
			}
		}()
		err := next(ctx, req, resp)
		panicked = false
		return err
	}
}

func (i *recoverHandlerInterceptor) WrapUnaryHandler(next UnaryHandlerFunc) UnaryHandlerFunc {
	return func(ctx context.Context, request AnyRequest) (resp AnyResponse, retErr error) {
		panicked := true
		defer func() {
			if panicked {
				r := recover()
				// net/http checks for ErrAbortHandler with ==, so we should too.
				if r == http.ErrAbortHandler { //nolint:errorlint,goerr113
					panic(r) //nolint:forbidigo
				}
				retErr = i.handle(ctx, request.Spec(), request.Header(), r)
			}
		}()
		response, retErr := next(ctx, request)
		panicked = false
		return response, retErr
	}
}

func (i *recoverHandlerInterceptor) WrapStreamingHandler(next StreamingHandlerFunc) StreamingHandlerFunc {
	return func(ctx context.Context, conn StreamingHandlerConn) (retErr error) {
		panicked := true
		defer func() {
			if panicked {
				r := recover()
				// net/http checks for ErrAbortHandler with ==, so we should too.
				if r == http.ErrAbortHandler { //nolint:errorlint,goerr113
					panic(r) //nolint:forbidigo
				}
				retErr = i.handle(ctx, Spec{}, nil, r)
			}
		}()
		err := next(ctx, conn)
		panicked = false
		return err
	}
}
