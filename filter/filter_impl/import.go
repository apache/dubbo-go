package filter_impl

// This package is for being compatible with older dubbo-go, please refer to https://github.com/dubbogo/imports to see
// the recommended import ways.
// This package may be DEPRECATED OR REMOVED in the future.

import (
	_ "dubbo.apache.org/dubbo-go/v3/filter/accesslog"
	_ "dubbo.apache.org/dubbo-go/v3/filter/active"
	_ "dubbo.apache.org/dubbo-go/v3/filter/auth"
	_ "dubbo.apache.org/dubbo-go/v3/filter/echo"
	_ "dubbo.apache.org/dubbo-go/v3/filter/execlmt"
	_ "dubbo.apache.org/dubbo-go/v3/filter/generic"
	_ "dubbo.apache.org/dubbo-go/v3/filter/gshutdown"
	_ "dubbo.apache.org/dubbo-go/v3/filter/hystrix"
	_ "dubbo.apache.org/dubbo-go/v3/filter/metrics"
	_ "dubbo.apache.org/dubbo-go/v3/filter/seata"
	_ "dubbo.apache.org/dubbo-go/v3/filter/sentinel"
	_ "dubbo.apache.org/dubbo-go/v3/filter/token"
	_ "dubbo.apache.org/dubbo-go/v3/filter/tps"
	_ "dubbo.apache.org/dubbo-go/v3/filter/tracing"
)

func init() {}
