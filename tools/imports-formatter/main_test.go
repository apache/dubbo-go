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
