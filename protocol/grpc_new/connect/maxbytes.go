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

//go:build go1.19

package connect

import (
	"errors"
	"fmt"
	"net/http"
)

func asMaxBytesError(err error, tmpl string, args ...any) *Error {
	var maxBytesErr *http.MaxBytesError
	if ok := errors.As(err, &maxBytesErr); !ok {
		return nil
	}
	prefix := fmt.Sprintf(tmpl, args...)
	return errorf(CodeResourceExhausted, "%s: exceeded %d byte http.MaxBytesReader limit", prefix, maxBytesErr.Limit)
}
