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

package p2c

import (
	"testing"
)

func TestDefaultRndVariesOrderForTwoInvokers(t *testing.T) {
	first := map[int]bool{}
	for range 200 {
		i, j := defaultRnd(2)
		if i == j || i < 0 || i > 1 || j < 0 || j > 1 {
			t.Fatalf("defaultRnd(2) = (%d, %d), want two distinct indices in [0, 2)", i, j)
		}
		first[i] = true
	}
	if len(first) != 2 {
		t.Fatalf("first index took only the values %v, want both 0 and 1", first)
	}
}
