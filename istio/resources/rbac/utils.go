package rbac

import (
	"fmt"
	"strings"
)

func getHeader(target string, headers map[string]string) (string, bool) {
	target = strings.ToLower(target)
	if target == "source.principal" {
		target = ":source.principal"
	}

	if target == "source.ip" {
		target = ":source.ip"
	}

	if strings.HasPrefix(target, "request.auth.") {
		target = fmt.Sprintf(":%s", target)
	}

	v, ok := headers[target]
	return v, ok
}
