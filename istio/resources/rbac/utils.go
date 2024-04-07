package rbac

import (
	"fmt"
	"strings"

	"dubbo.apache.org/dubbo-go/v3/common/constant"
)

func getHeader(target string, headers map[string]string) (string, bool) {
	if target == "source.principal" {
		target = ":source.principal"
	}

	if target == "source.ip" {
		target = ":source.ip"
	}

	if strings.HasPrefix(target, "request.auth.") {
		target = fmt.Sprintf(":%s", target)
	}
	
	switch strings.ToLower(target) {
	case constant.HttpHeaderXSchemeName:
		v, ok := headers[constant.HttpHeaderXSchemeName]
		return v, ok
	case constant.HttpHeaderXMethodName:
		v, ok := headers[constant.HttpHeaderXMethodName]
		return v, ok
	case constant.HttpHeaderXSourceIp:
		v, ok := headers[constant.HttpHeaderXSourceIp]
		return v, ok
	case constant.HttpHeaderXSpiffeName:
		v, ok := headers[constant.HttpHeaderXSpiffeName]
		return v, ok
	case constant.HttpHeaderXSourcePrincipal:
		v, ok := headers[constant.HttpHeaderXSourcePrincipal]
		return v, ok
	case constant.HttpHeaderXPathName:
		v, ok := headers[constant.HttpHeaderXPathName]
		return v, ok
	default:
		return "", false
	}
}
