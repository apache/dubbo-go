# Filter

## Getting Started

Recommended Way: import what you needs, see also [dubbo-go/imports](https://github.com/dubbogo/imports).

```go
package demo

// use echo and generic filters
import _ "dubbo.apache.org/dubbo-go/v3/filter/echo"
import _ "dubbo.apache.org/dubbo-go/v3/filter/generic"
```

Legacy way: import all filters by one line.

```go
package demo

import _ "dubbo.apache.org/dubbo-go/v3/filter/filter_impl"
```

## Contents

- accesslog: Access Log Filter(https://github.com/apache/dubbo-go/pull/214)
- active
- auth: Auth/Sign Filter(https://github.com/apache/dubbo-go/pull/323)
- echo: Echo Health Check Filter
- execlmt: Execute Limit Filter(https://github.com/apache/dubbo-go/pull/246)
- generic: Generic Filter(https://github.com/apache/dubbo-go/pull/291)
- gshutdown: Graceful Shutdown Filter
- hystrix: Moved to [dubbo-go-extensions](https://github.com/apache/dubbo-go-extensions/tree/main/filter/hystrix)
- metrics: Metrics Filter(https://github.com/apache/dubbo-go/pull/342)
- seata: Seata Filter
- sentinel: Sentinel Filter
- token: Token Filter(https://github.com/apache/dubbo-go/pull/202)
- tps: Tps Limit Filter(https://github.com/apache/dubbo-go/pull/237)
- tracing: Tracing Filter(https://github.com/apache/dubbo-go/pull/335)

## Java overloaded methods

Go does not support Java-style method overloading. When a Java interface has
multiple methods with the same name and different arguments, expose one Go
method for that Java name and dispatch by argument count or argument type inside
the method.

```go
import (
	"context"
	"fmt"
)

type CommentServiceProvider struct{}

func (p *CommentServiceProvider) IsForbidsSpeak(ctx context.Context, params []any) (any, error) {
	switch len(params) {
	case 2:
		bookID, userID := params[0], params[1]
		// Handle: isForbidsSpeak(Long bookId, Long userId)
		_, _ = bookID, userID
	case 3:
		bookID, userID, ip := params[0], params[1], params[2]
		// Handle: isForbidsSpeak(Long bookId, Long userId, String ip)
		_, _, _ = bookID, userID, ip
	default:
		return nil, fmt.Errorf("unsupported isForbidsSpeak argument count: %d", len(params))
	}
	return true, nil
}

func (p *CommentServiceProvider) MethodMapper() map[string]string {
	return map[string]string{
		"IsForbidsSpeak": "isForbidsSpeak",
	}
}
```

For stricter dispatch, check both the argument count and the concrete argument
types. This keeps the exported Go service shape explicit while still matching
the overloaded Java method name through `MethodMapper`.
