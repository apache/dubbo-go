# Server-Side Adaptive Service

This document covers the provider-side adaptive service filter, method-level
limiter, and Hill Climbing adaptive concurrency algorithm.

## Related Files

Provider-side adaptive service:

- `filter/adaptivesvc/filter.go`
- `filter/adaptivesvc/limiter_mapper.go`

Provider-side limiter:

- `filter/adaptivesvc/limiter/limiter.go`
- `filter/adaptivesvc/limiter/hill_climbing.go`
- `filter/adaptivesvc/limiter/utils.go`

Configuration and service export:

- `config/provider_config.go`
- `config/service_config.go`
- `server/options.go`
- `server/action.go`

Shared constants:

- `common/constant/key.go`

## Configuration and Usage

### Configuration Files

Provider-side config fields:

```yaml
dubbo:
  provider:
    adaptive-service: true
    adaptive-service-verbose: false
```

`adaptive-service` enables the provider-side adaptive service filter. During
service export, the service config appends the provider filter key `padasvc` to
`service.filter`, so provider requests can pass through
`adaptiveServiceProviderFilter`.

`adaptive-service-verbose` enables verbose limiter logs. It is valid only when
`adaptive-service` is also enabled.

### API Options

Provider-side server options:

```go
server.WithServerAdaptiveService()
server.WithServerAdaptiveServiceVerbose()
```

`WithServerAdaptiveService()` enables the provider adaptive service filter.
`WithServerAdaptiveServiceVerbose()` enables verbose limiter logs and should be
used only together with adaptive service.

### Required Imports

The provider filter is registered through a package `init()` function. Applications
need to import the all-in-one imports package or import the provider filter
directly:

```go
import _ "dubbo.apache.org/dubbo-go/v3/imports"

// or
import _ "dubbo.apache.org/dubbo-go/v3/filter/adaptivesvc"
```

## Architecture

```text
provider side
├── filter registration and creation
│   ├── package init()
│   │   └── extension.SetFilter("padasvc", newAdaptiveServiceProviderFilter)
│   ├── service.filter contains "padasvc"
│   │   └── ProtocolFilterWrapper builds provider filter chain
│   └── newAdaptiveServiceProviderFilter()
│       └── lazy singleton: adaptiveServiceProviderFilter instance
│
└── adaptiveServiceProviderFilter
    ├── Invoke()
    │   ├── check invocation attachment
    │   │   └── adaptive-service.enabled == "1"
    │   ├── get method-level limiter
    │   │   └── limiterMapperSingleton
    │   │       ├── eager singleton, created during package init
    │   │       ├── mapper: map[service + method]limiter.Limiter
    │   │       └── create limiter if no limiter exists for the method
    │   ├── limiter.Acquire()
    │   ├── save Updater into invocation attribute
    │   │   └── adaptive-service.updater
    │   └── invoke business logic
    │
    └── OnResponse()
        ├── get Updater from invocation attribute
        ├── updater.DoUpdate()
        ├── get current method-level limiter from limiterMapperSingleton
        └── write response attachments
            ├── adaptive-service.remaining
            └── adaptive-service.inflight
```

## Provider Filter Flow

The provider-side filter is the outer layer in Dubbo's provider filter chain. It
does not store limiter state directly.

### Filter Registration

During package initialization, `filter/adaptivesvc/filter.go` registers the
provider filter factory under the filter key `padasvc`:

```go
extension.SetFilter(constant.AdaptiveServiceProviderFilterKey, newAdaptiveServiceProviderFilter)
```

When provider adaptive service is enabled, service export appends `padasvc` to
`service.filter`. Dubbo's provider filter wrapper then resolves `padasvc` through
the extension registry and builds `adaptiveServiceProviderFilter` into the
provider invocation chain.

### Request Handling

`Invoke()` checks whether the consumer request carries:

```text
adaptive-service.enabled = "1"
```

If the attachment is absent, the filter invokes business logic directly. If it is
present, the filter gets the current method limiter from
`limiterMapperSingleton`, then calls `Acquire()` before invoking business logic.

The limiter mapper uses method-level granularity, for example:

```text
UserService.GetUser -> limiter A
UserService.ListUser -> limiter B
OrderService.GetUser -> limiter C
```

The provider filter stores the limiter updater in a local invocation attribute:

```text
adaptive-service.updater
```

`OnResponse()` reads this updater, calls `DoUpdate()`, gets the current limiter
again, and writes provider status into response attachments:

```text
adaptive-service.remaining
adaptive-service.inflight
```

The provider filter instance is a lazy singleton. `limiterMapperSingleton` is
created during package initialization and owns the method-to-limiter map. A
concrete method limiter is created only when that method is first invoked through
adaptive service.

Currently, the adaptive service provider-side limiter only supports
`HillClimbingLimiter`.

## Attachments and Attributes

| Key | Direction or Scope | Meaning |
| --- | --- | --- |
| `adaptive-service.enabled` | consumer -> provider | Marks this request as participating in adaptive service. The value is `1`. |
| `adaptive-service.remaining` | provider -> consumer | Provider method limiter's remaining capacity after the request is processed. |
| `adaptive-service.inflight` | provider -> consumer | Provider method limiter's current in-flight request count. |
| `adaptive-service.updater` | provider process only | Local invocation attribute storing the limiter updater returned by `Acquire()`. |

## Hill Climbing Algorithm

The Hill Climbing limiter adjusts the allowed concurrency for each service method.
It tries to increase the limit while the method is healthy, and decrease the limit
when higher concurrency no longer improves throughput or starts hurting latency.

### Flow

```text
Acquire()
├── calculate remaining = limitation - inflight
├── reject if remaining == 0
└── create HillClimbingUpdater
    ├── record request start time
    └── increase inflight

DoUpdate()
├── calculate request RTT
├── getOption(rtt, inflight)
│   ├── update current sampling-window metrics
│   └── return adjustment option
│       ├── ExtendPlus
│       ├── Extend
│       ├── DoNothing
│       ├── Shrink
│       └── ShrinkPlus
├── adjustLimitation(option)
│   └── update limitation according to the option
└── decrease inflight
```

### `getOption()`

- `getOption()` runs during `DoUpdate()` after the request RTT is calculated. If
  the current sampling window is still active, it only updates `transactionNum`
  and `rttAvg`, then returns `DoNothing`. The limiter does not change
  `limitation` in the middle of a sampling window.

- When a sampling window ends, it calculates
  `tps = 1000 * transactionNum / updateInterval(milliseconds)` and
  `maxCapacity = transactionNum * updateInterval(milliseconds) / rttAvg`, then
  compares the current window with the best historical metrics.

- It chooses an extend option when there is no historical best RTT yet, or when
  `maxCapacity * 1.5 > limitation`. This means the limiter believes the current
  method can accept a higher concurrency limit.

- It updates the best historical metrics when `tps > bestTPS`. If `tps` is not
  better than the historical best, `shouldShrink()` may choose a shrink option.
  Shrinking requires a significant capacity drop compared with `bestMaxCapacity`
  and enough RTT degradation compared with `bestRTTAvg`.

- It returns `DoNothing` when the sampling window is still active, or when a
  completed sampling window does not meet either the extend or shrink conditions.

### `adjustLimitation()`

- `DoNothing` does not change `limitation`.

- Extend and shrink each have two variants. `ExtendPlus` / `ShrinkPlus` are
  stronger adjustments used during the initial `1s` radical period.
  `Extend` / `Shrink` are softer adjustments used after the limiter moves to a
  longer update interval.

- `alpha` and `beta` are adjustment step sizes derived from the current
  `limitation`. `alpha = 1.5 * ln(limitation)` is the larger step and is used by
  `ExtendPlus` / `ShrinkPlus`; `beta = 0.8 * ln(limitation)` is the smaller step
  and is used by `Extend` / `Shrink`.

- For extension, the limiter increases the current `limitation`: `ExtendPlus`
  adds `alpha / logUpdateInterval`, and `Extend` adds
  `beta / logUpdateInterval`.

- For shrinking, the limiter moves back from the best historical limit:
  `ShrinkPlus` sets `limitation` to `bestLimitation - alpha / logUpdateInterval`,
  and `Shrink` sets it to `bestLimitation - beta / logUpdateInterval`.

- `logUpdateInterval = max(1, log2(updateInterval / 1s))` makes adjustments
  smaller when the sampling window becomes longer. The final `limitation` is
  clamped to `[1, 500]`.

### Known Limitation

- In a workload where request latency rises sharply, for example from `20ms` to
  `100ms` or `500ms`, the limiter may not reduce the concurrency limit
  immediately or significantly.

- This can happen because shrinking depends on sampling-window metrics such as
  TPS and RTT, plus historical best metrics, not on a single RTT increase. If TPS
  does not clearly get worse, or the shrink threshold is not reached, the limiter
  can keep the limit close to the previous level.

- Even when shrinking is triggered, the decrease is gradual rather than
  proportional to the RTT increase, so the limiter may look insensitive in
  step-latency tests.
