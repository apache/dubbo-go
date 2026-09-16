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

package options

import "testing"

func TestGenerateUseHTTPRules(t *testing.T) {
	got, err := Generate("format=json,use-http-rules=true")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got.Format != "json" || !got.UseHTTPRules {
		t.Fatalf("options = %#v", got)
	}

	got, err = Generate("use-http-rules=false")
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if got.UseHTTPRules {
		t.Fatal("use-http-rules=false should disable HTTP rules")
	}
}

func TestGenerateRejectsInvalidUseHTTPRules(t *testing.T) {
	if _, err := Generate("use-http-rules=maybe"); err == nil {
		t.Fatal("Generate() should reject an invalid boolean")
	}
}
