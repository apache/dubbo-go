package config

import (
	"testing"
)

import (
	"github.com/stretchr/testify/assert"
)

func TestMergeValue(t *testing.T) {
	str := mergeValue("", "", "a,b")
	assert.Equal(t, "a,b", str)

	str = mergeValue("c,d", "e,f", "a,b")
	assert.Equal(t, "a,b,c,d,e,f", str)

	str = mergeValue("c,d", "e,d,f", "a,b")
	assert.Equal(t, "a,b,c,d,e,d,f", str)

	str = mergeValue("c,default,d", "-c,-a,e,f", "a,b")
	assert.Equal(t, "b,d,e,f", str)

	str = mergeValue("", "default,-b,e,f", "a,b")
	assert.Equal(t, "a,e,f", str)
}
