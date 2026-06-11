# Filter

## Getting Started

Recommended Way: import only the filters you need. See also [dubbo-go/imports](https://github.com/dubbogo/imports).

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
- adaptivesvc: Adaptive Service Filter
- auth: Auth/Sign Filter(https://github.com/apache/dubbo-go/pull/323)
- echo: Echo Health Check Filter
- exec_limit: Execute Limit Filter(https://github.com/apache/dubbo-go/pull/246)
- generic: Generic Filter(https://github.com/apache/dubbo-go/pull/291)
- graceful_shutdown: Graceful Shutdown Filter
- hystrix: Moved to [dubbo-go-extensions](https://github.com/apache/dubbo-go-extensions/tree/main/filter/hystrix)
- metrics: Metrics Filter(https://github.com/apache/dubbo-go/pull/342)
- otel/trace: OpenTelemetry Tracing Filter
- polaris/limit: Polaris Rate Limit Filter
- seata: Seata Filter
- sentinel: Sentinel Filter
- token: Token Filter(https://github.com/apache/dubbo-go/pull/202)
- tps: Tps Limit Filter(https://github.com/apache/dubbo-go/pull/237)
- tracing: Tracing Filter(https://github.com/apache/dubbo-go/pull/335)

## Generic response attachments

In dubbo-go 3.x, `GenericService2` is no longer the response attachment access
point used by older 1.5.x applications. For Triple generic calls, provider-side
result attachments are written back as response trailers and can be read from
the call context after the generic invocation returns.

Initialize the context with Triple outgoing metadata before the call. This also
enables the client-side Triple transport to copy response trailers back into
the context.

```go
package demo

import (
	"context"
	"net/http"

	hessian "github.com/apache/dubbo-go-hessian2"

	"dubbo.apache.org/dubbo-go/v3/filter/generic"
	triple "dubbo.apache.org/dubbo-go/v3/protocol/triple/triple_protocol"
)

func invokeWithResponseAttachments(genericService *generic.GenericService) (http.Header, error) {
	ctx := triple.NewOutgoingContext(context.Background(), http.Header{})

	_, err := genericService.Invoke(ctx, "echo", []string{"java.lang.String"}, []hessian.Object{"hello"})
	if err != nil {
		return nil, err
	}

	trailers, ok := triple.FromIncomingContext(ctx)
	if !ok {
		return nil, nil
	}
	return trailers, nil
}
```

On the provider side, attachments set on the invocation result are propagated to
the Triple response trailers for unary calls.
