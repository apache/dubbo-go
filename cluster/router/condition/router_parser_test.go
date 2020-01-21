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
	assert.Equal(t, "application", rule.Scope)
	assert.True(t, rule.Runtime)
	assert.Equal(t, false, rule.Force)
	assert.Equal(t, testyml, rule.RawRule)
	assert.True(t, true, rule.Valid)
	assert.Equal(t, false, rule.Enabled)
	assert.Equal(t, false, rule.Dynamic)
	assert.Equal(t, "", rule.Key)
}
