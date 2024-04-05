package rbac

import (
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func getHeader(target string, headers map[string]string) (string, bool) {
	switch strings.ToLower(target) {
	case constant.HttpHeaderXSchemeName:
		v, ok := headers[constant.HttpHeaderXSchemeName]
		return v, ok
	case constant.HttpHeaderXMethodName:
		v, ok := headers[constant.HttpHeaderXMethodName]
		return v, ok
	case constant.HttpHeaderXRemoteIp:
		v, ok := headers[constant.HttpHeaderXRemoteIp]
		return v, ok
	case constant.HttpHeaderXSpiffeName:
		v, ok := headers[constant.HttpHeaderXSpiffeName]
		return v, ok
	case constant.HttpHeaderXPathName:
		v, ok := headers[constant.HttpHeaderXPathName]
		return v, ok
	default:
		return "", false
	}
}
