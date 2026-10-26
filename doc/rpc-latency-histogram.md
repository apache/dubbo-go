# RPC latency histograms

RPC latency histograms are optional and disabled by default. They are independent
of sliding-window aggregation and leave existing RT, quantile, request and error
metrics unchanged. Enable metrics and histograms explicitly:

```yaml
dubbo:
  metrics:
    enable: true
    histogram:
      enabled: true
      buckets: [0.1, 0.3, 0.5, 1, 3, 5, 10]
```

The equivalent instance option is:

```go
dubbo.WithMetrics(
    metrics.WithEnabled(),
    metrics.WithHistogramEnabled(),
    metrics.WithHistogramBuckets([]float64{0.1, 0.3, 0.5, 1, 3, 5, 10}),
)
```

`WithHistogramEnabled` does not implicitly enable metrics. Configuring buckets
alone does not enable histograms. Configuration is applied during Metrics
initialization; changing bucket boundaries requires restarting the process.

## Exported contract

| Side | Metric family |
| --- | --- |
| Provider | `dubbo_provider_rt_milliseconds_histogram_seconds` |
| Consumer | `dubbo_consumer_rt_milliseconds_histogram_seconds` |

The historical `milliseconds` segment matches the existing Dubbo Java metric
names. **Observations, bucket boundaries and `_sum` are in seconds**, recorded
from `time.Duration.Seconds()` without truncating sub-millisecond durations.
Existing RT metrics retain their millisecond recording path.

Each family exports `_bucket` (including `le="+Inf"`), `_sum` and `_count`.
Successful and failed completed invocation events both contribute. Provider and
consumer observations describe different invocation boundaries; do not combine
both sides as one latency distribution. The measurement follows the existing
Metrics Filter invocation boundary, not an entire streaming session or a business
transaction spanning retries.

Labels are exactly the existing RPC RT labels: `application_name`,
`application_version`, `hostname`, `ip`, `interface`, `method`, `group` and
`version`. No trace ID, request ID or additional `side` label is introduced.

Finite bounds must be positive, finite and strictly increasing, with no
duplicates. Missing bounds use `[0.1, 0.3, 0.5, 1, 3, 5, 10]` seconds. Invalid
bounds fall back as a whole to these defaults with a startup warning. These
defaults match Java's seven default finite latency boundaries; custom Java
configuration and generated percentile buckets may differ.

The Prometheus Registry implements the optional
`metrics.ConfigurableHistogramRegistry` interface. Other registries remain
source-compatible. If an enabled RPC histogram cannot use this capability, a
startup warning is emitted and only the new histograms are disabled. Existing
metrics continue to work; custom bucket configuration is not silently ignored.
The original `MetricRegistry.Histogram` retains the registry's ordinary defaults.

Bucket layout is fixed by the first registration of each metric name, across all
label sets. Do not register the same family with inconsistent bounds or label
names. Options and configuration slices are copied, not retained from callers.

## Prometheus queries

Provider heatmap input (bucket rates, not a quantile):

```promql
sum by (le) (
  rate(dubbo_provider_rt_milliseconds_histogram_seconds_bucket{
    application_name="$application"
  }[$__rate_interval])
)
```

Provider application-level P99:

```promql
histogram_quantile(0.99,
  sum by (le) (
    rate(dubbo_provider_rt_milliseconds_histogram_seconds_bucket{
      application_name="$application"
    }[$__rate_interval])
  )
)
```

Replace `provider` with `consumer` for the consumer side. Use seconds as the
display unit. Aggregation requires compatible bucket boundaries and units across
all selected processes. Use interface, method, group and version filters to
select the intended service identity. Do not average local P99 values and label
that result a global P99, or replace unavailable histograms with zero.

Histogram counters are cumulative and do not reset on scrape or when traffic
stops. Query `rate`/`increase` over a window. An idle or unavailable distribution
does not imply zero latency. Process restarts reset counters, which `rate` handles.

Seven finite bounds yield ten classic time series per RPC label combination:
eight buckets including `+Inf`, plus sum and count. Account for this cardinality
before enabling histograms across many methods and instances.

## Validation and limits

```sh
go test ./metrics -run 'TestHistogram' -count=1
go test ./metrics/prometheus -run 'TestHistogramWithOptions' -count=1
go test ./metrics/rpc ./metrics/rpc/integration -count=1
go test . -run 'TestHistogram' -count=1
go test -race ./metrics/prometheus -run 'TestHistogramWithOptions' -count=1
```

Tests cover custom and invalid bounds, precision, concurrency, configuration
loading and cloning, both sides, success and failure, preservation of old metrics,
and repeated scrapes. The integration test exercises the real Metrics Filter,
event bus, RPC collector and Prometheus HTTP handler with an in-process invoker;
it does not prove a network RPC or Kubernetes deployment.

This feature uses the existing Metrics event bus. Queue overflow can still drop
events; histograms do not fix that existing limitation. It does not change error
classification, sliding-window refresh, native histograms, tracing, spanmetrics,
or Dubbo Admin dashboards. Prometheus still needs to scrape the Go exporter;
adding these metrics does not configure Kubernetes Service discovery or scraping.
