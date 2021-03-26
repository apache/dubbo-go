package match_judger

import (
	"github.com/apache/dubbo-go/config"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestListDoubleMatchJudger_Judge(t *testing.T) {
	assert.True(t, newListDoubleMatchJudger(&config.ListDoubleMatch{
		Oneof: []*config.DoubleMatch{
			{
				Exact: 3.14,
			},
			{
				Range: &config.DoubleRangeMatch{
					Start: 1.5,
					End:   1.9,
				},
			},
			{
				Exact: 1.3,
			},
		},
	}).Judge(1.3))

	assert.False(t, newListDoubleMatchJudger(&config.ListDoubleMatch{
		Oneof: []*config.DoubleMatch{
			{
				Exact: 3.14,
			},
			{
				Range: &config.DoubleRangeMatch{
					Start: 1.5,
					End:   1.9,
				},
			},
			{
				Exact: 1.2,
			},
		},
	}).Judge(1.3))

	assert.True(t, newListDoubleMatchJudger(&config.ListDoubleMatch{
		Oneof: []*config.DoubleMatch{
			{
				Exact: 3.14,
			},
			{
				Range: &config.DoubleRangeMatch{
					Start: 1.2,
					End:   1.9,
				},
			},
			{
				Exact: 1.4,
			},
		},
	}).Judge(1.3))

	assert.False(t, newListDoubleMatchJudger(&config.ListDoubleMatch{
		Oneof: []*config.DoubleMatch{
			{
				Exact: 3.14,
			},
			{
				Range: &config.DoubleRangeMatch{
					Start: 1.5,
					End:   1.9,
				},
			},
			{
				Exact: 1.0,
			},
		},
	}).Judge(1.3))
}
