package match_judger

import (
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol/invocation"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestAttachmentMatchJudger(t *testing.T) {
	dubboCtxMap := make(map[string]*config.StringMatch)
	dubboIvkMap := make(map[string]interface{})
	dubboCtxMap["test-key"] = &config.StringMatch{
		Exact: "abc",
	}
	dubboIvkMap["test-key"] = "abc"
	assert.True(t, NewAttachmentMatchJudger(&config.DubboAttachmentMatch{
		DubboContext: dubboCtxMap,
	}).Judge(invocation.NewRPCInvocation("method", nil, dubboIvkMap)))

	dubboIvkMap["test-key"] = "abd"
	assert.False(t, NewAttachmentMatchJudger(&config.DubboAttachmentMatch{
		DubboContext: dubboCtxMap,
	}).Judge(invocation.NewRPCInvocation("method", nil, dubboIvkMap)))

}
