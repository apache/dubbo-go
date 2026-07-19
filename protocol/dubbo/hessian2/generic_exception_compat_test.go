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

package hessian2_test

import (
	"testing"
)

import (
	dubbohessian "dubbo.apache.org/dubbo-go/v3/protocol/dubbo/hessian2"
)

// TestGenericExceptionCompat verifies that the backward-compatible GenericException
// type alias and ToGenericException function are still accessible from the original
// import path (dubbo.apache.org/dubbo-go/v3/protocol/dubbo/hessian2).
//
// This test uses an external package (hessian2_test) so it validates the public API
// surface as an external consumer would.
func TestGenericExceptionCompat(t *testing.T) {
	// 1) Type alias — should compile and create a valid struct literal
	ge := dubbohessian.GenericException{
		ExceptionClass:   "com.example.TestException",
		ExceptionMessage: "test error",
	}
	if ge.Error() == "" {
		t.Fatal("GenericException.Error() should not be empty")
	}

	// 2) ToGenericException — Deprecated forwarding function
	result, ok := dubbohessian.ToGenericException(ge)
	if !ok || result == nil {
		t.Fatal("ToGenericException should return (non-nil, true) for a valid GenericException")
	}
	if result.ExceptionClass != "com.example.TestException" {
		t.Fatalf("unexpected ExceptionClass: got %q, want %q",
			result.ExceptionClass, "com.example.TestException")
	}

	// 3) ToGenericException with a string (legacy path)
	result2, ok2 := dubbohessian.ToGenericException("java exception: com.example.Err - something went wrong")
	if !ok2 || result2 == nil {
		t.Fatal("ToGenericException should handle legacy string format")
	}
}
