package condition

import (
	"testing"
)
import (
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	testyml := `
scope: application
runtime: true
force: false
conditions:
  - >
    method!=sayHello =>
  - >
    ip=127.0.0.1
    =>
    1.1.1.1`
	rule, e := Parse(testyml)

	assert.Nil(t, e)
	assert.NotNil(t, rule)
	assert.Equal(t, 2, len(rule.Conditions))
}
