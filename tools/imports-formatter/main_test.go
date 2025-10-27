// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"testing"
)

import (
	"github.com/magiconair/properties/assert"
)

func TestMergeImports(t *testing.T) {
	imports := make(map[string][]string)
	imports["common"] = []string{"common"}
	imports["common/constant"] = []string{"common/constant"}
	imports["common/extension"] = []string{"common/extension"}
	imports["common/logger"] = []string{"common/logger"}
	imports["registry"] = []string{"registry"}

	mImports := mergeImports(imports)
	for i := 0; i < 100; i++ {
		assert.Equal(t, 2, len(mImports))
		commonImports := mImports["common"]
		if len(commonImports) != 4 {
			t.Error(mImports)
		}
	}
}
