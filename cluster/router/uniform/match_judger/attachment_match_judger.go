package match_judger

import (
	"github.com/apache/dubbo-go/config"
	"github.com/apache/dubbo-go/protocol"
)

type AttachmentMatchJudger struct {
	config.DubboAttachmentMatch
}

func (amj *AttachmentMatchJudger) Judge(invocation protocol.Invocation) bool {
	invAttaMap := invocation.Attachments()
	if amj.EagleeyeContext != nil {
		for k, v := range amj.EagleeyeContext {
			invAttaValue, ok := invAttaMap[k]
			if !ok {
				if v.Empty == "" {
					return false
				}
			}
			// exist this key
			str, ok := invAttaValue.(string)
			if !ok {
				return false
			}
			strJudger := NewStringMatchJudger(v)
			if !strJudger.Judge(str) {
				return false
			}
		}
	}

	if amj.DubboContext != nil {
		for k, v := range amj.DubboContext {
			invAttaValue, ok := invAttaMap[k]
			if !ok {
				if v.Empty == "" {
					return false
				}
			}
			// exist this key
			str, ok := invAttaValue.(string)
			if !ok {
				return false
			}
			strJudger := NewStringMatchJudger(v)
			if !strJudger.Judge(str) {
				return false
			}
		}
	}

	return true
}

func NewAttachmentMatchJudger(matchConf *config.DubboAttachmentMatch) *AttachmentMatchJudger {
	return &AttachmentMatchJudger{
		DubboAttachmentMatch: *matchConf,
	}
}
