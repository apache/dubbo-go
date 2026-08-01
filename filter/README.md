# Filter

Dubbo-go filters are interceptor-style extensions around RPC invocation.
They are typically used to add cross-cutting behavior such as authentication,
rate limiting, logging, tracing, metrics, generic invocation adaptation, and
graceful shutdown handling.

## Getting Started

The recommended approach is to import only the filters your application needs.

```go
package demo

import _ "dubbo.apache.org/dubbo-go/v3/filter/echo"
import _ "dubbo.apache.org/dubbo-go/v3/filter/generic"
```

If you prefer the legacy all-in-one import, you can still use:

```go
package demo

import _ "dubbo.apache.org/dubbo-go/v3/filter/filter_impl"
```

## Where Filters Apply

Dubbo-go distinguishes between two filter chains:

- `service.filter`: provider-side filters
- `reference.filter`: consumer-side filters

The built-in defaults are defined in `common/constant/default.go`.

## Built-in Filters

### Core

- `echo`: echo health check support
- `generic`: consumer-side generic invocation filter
- `generic_service`: provider-side generic service filter
- `graceful_shutdown`: graceful shutdown behavior for provider and consumer paths

### Traffic Control

- `active`: active request counting
- `exec_limit`: provider-side execute concurrency limiting
- `tps`: TPS limiting
- `adaptivesvc`: adaptive service filter
- `polaris/limit`: Polaris rate limiting
- `sentinel`: Sentinel integration for provider and consumer traffic protection

### Security

- `token`: token-based request validation
- `auth`: request signing and authentication
- `seata`: Seata distributed transaction propagation

### Observability

- `accesslog`: invocation access logging
- `metrics`: metrics collection
- `tracing`: tracing integration
- `otel/trace`: OpenTelemetry tracing integration

### Extension Note

- `hystrix`: moved to [dubbo-go-extensions](https://github.com/apache/dubbo-go-extensions/tree/main/filter/hystrix)

## Common Usage

In practice, filters are usually enabled through application config instead of
being called directly from business code. Typical examples:

- enable a provider filter through `service.filter`
- enable a consumer filter through `reference.filter`
- import the filter package so it registers itself via `extension.SetFilter(...)`

## Notes

- Importing a filter package registers it; the filter still needs to be enabled
  by the corresponding consumer/provider configuration path when applicable.
- Some filter names differ slightly from their package directory because the
  runtime key is defined in `common/constant/key.go`.
- Filters under subdirectories such as `otel/trace` and `polaris/limit` are
  still ordinary Dubbo-go filters; the subdirectory just reflects feature
  grouping.
