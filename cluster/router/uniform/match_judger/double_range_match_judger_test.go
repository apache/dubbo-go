package match_judger

import (
	"github.com/apache/dubbo-go/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDoubleRangeMatchJudger(t *testing.T) {
	assert.True(t, newDoubleRangeMatchJudger(&config.DoubleRangeMatch{
		Start: 1.0,
		End:   1.5,
	}).Judge(1.3))

	assert.False(t, newDoubleRangeMatchJudger(&config.DoubleRangeMatch{
		Start: 1.0,
		End:   1.5,
	}).Judge(1.9))

	assert.False(t, newDoubleRangeMatchJudger(&config.DoubleRangeMatch{
		Start: 1.0,
		End:   1.5,
	}).Judge(0.9))
}
