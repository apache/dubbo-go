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
- hystrix: Hystric Filter(https://github.com/apache/dubbo-go/pull/133)
- metrics: Metrics Filter(https://github.com/apache/dubbo-go/pull/342)
- seata: Seata Filter
- sentinel: Sentinel Filter
- token: Token Filter(https://github.com/apache/dubbo-go/pull/202)
- tps: Tps Limit Filter(https://github.com/apache/dubbo-go/pull/237)
- tracing: Tracing Filter(https://github.com/apache/dubbo-go/pull/335)