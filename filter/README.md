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
