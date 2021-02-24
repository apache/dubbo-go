package match_judger

import (
	"github.com/apache/dubbo-go/common"
)

func JudgeUrlLabel(url *common.URL, labels map[string]string) bool {
	for k, v := range labels {
		if url.GetParam(k, "") != v {
			return false
		}
	}
	return true
}
