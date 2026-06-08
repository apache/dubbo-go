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

## Passing HTTP Headers

Dubbo filters operate on Dubbo invocations, so a client filter cannot read
HTTP framework headers directly. If an HTTP gateway such as Gin needs to pass
headers to a Dubbo provider, copy the required headers in the HTTP layer and
make them available to a custom Dubbo client filter through `context.Context`.
The filter can then copy those values into Dubbo attachments.

```go
type requestIDKey struct{}

func handler(c *gin.Context) {
	req := &UserRequest{}
	ctx := context.WithValue(c.Request.Context(), requestIDKey{}, c.GetHeader("X-Request-ID"))

	resp, err := userProvider.GetUser(ctx, req)
	// handle resp and err
}
```

Custom client filters can also add or normalize attachments on the invocation:

```go
func (f *MyClientFilter) Invoke(ctx context.Context, invoker protocol.Invoker, invocation protocol.Invocation) protocol.Result {
	if requestID, ok := ctx.Value(requestIDKey{}).(string); ok && requestID != "" {
		invocation.SetAttachment("x-request-id", requestID)
	}
	return invoker.Invoke(ctx, invocation)
}
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
