# RPC failure origins and completion metrics

This extends local Network/Codec attribution for issue #3774. It does not
implement early business completion, change wire statuses, or redesign retry
and callback contracts.

## Data path

1. A confirmed serializer or transport boundary preserves a local origin with
   `common/rpcerror.Mark`. Its error text and cause chain are retained.
2. The Metrics Filter creates one Observation and snapshots the application,
   role, interface, method, group and version for each local invocation attempt.
3. Dubbo registers that Observation on PendingResponse before publishing the
   request, then binds it to the actual Getty Session before sending.
4. For a complete, validated non-event response frame, a decoder failure can
   carry the response ID. The Getty adapter records its origin only when both
   the pending request and original Session match. It does not remove the
   request, close Done, change the returned error, or invoke a user callback.
5. Synchronous Invoke return freezes the monitoring snapshot. Callback-based
   calls hand completion to the real response, with a buffered fast-response
   state to handle a callback arriving before Invoke returns. A synchronous
   submission error wins over that buffered response.
6. `metrics.Publish` synchronously invokes the RPC recorder, which increments
   total failure and exactly one category before delivering best-effort channel
   notifications. No transport or codec increments counters directly.

A successful result remains successful even if sidecar evidence exists.
Observation freezing is protected by a mutex. Response, cancellation and
monitoring timeout compete for one monitoring terminal state. Evidence arriving
after that state cannot change an already published event. Retries have
independent observations: a failed local attempt followed by a successful
attempt is not one successful metric event and is not one permanent failure.

## Classification

Every failed call increments `requests_failed_total` and exactly one category.
The category is chosen in this order (see `classifyError`):

1. Triple/gRPC DeadlineExceeded, ResourceExhausted, PermissionDenied and
   BizError keep their existing priority.
2. Confirmed local Network or Codec evidence (`rpcerror.Origin`). It is the
   root cause, so it wins over a symptom such as the read timeout that follows
   an undecodable response.
3. A producer-declared kind (`rpcerror.Kind`, see
   [Failure taxonomy](#failure-taxonomy-3337)).
4. An unmarked Unavailable remains ServiceUnavailable.

Plain Internal/InvalidArgument, error text, or a Session close alone do not
prove any category. Conflicting evidence remains Unknown.

Network evidence currently requires a `net.OpError` at the trusted Getty write
or Triple HTTP Do boundary; cancellation/deadline context errors are excluded.
Both the Triple unary fast path and duplex path preserve the evidence after
existing HTTP error normalization. Generic Getty connection setup timeouts
that have already lost the original dial error are not relabeled Network.

Codec evidence covers Dubbo serializer failures and Triple Marshal,
MarshalAppend, Unmarshal, Envelope and unary paths. A successful codec fallback
does not produce a failure. Compression and decompression are not newly
classified by this change.

## Exported metrics

For each `{side}` in `provider`, `consumer`:

| Metric | Type | Meaning |
| --- | --- | --- |
| `dubbo_{side}_requests_failed_network_total` | Counter | Confirmed Network failures |
| `dubbo_{side}_requests_failed_codec_total` | Counter | Confirmed Codec failures |
| `dubbo_{side}_requests_failed_network_total_aggregate` | Gauge | Network failures in the existing local window |
| `dubbo_{side}_requests_failed_codec_total_aggregate` | Gauge | Codec failures in the existing local window |

These use the same labels as the existing RPC metrics (see
[Label contract](#label-contract-3337)). There are no request-ID, Session, exception text
or other high-cardinality labels. Total failure is incremented once alongside
one category; Network/Codec failures do not also increment Unknown.

Framework services exported by dubbo-go itself, the gRPC health check
(`grpc.health.v1.Health`) and server reflection, are not application RPCs. The
provider runs them without the metrics filter and the OTel server trace filter,
as Dubbo Java runs its Triple builtin services outside the provider filter
chain, so health watch streams and probes do not distort provider RT and
failure rates. A dubbo-go client that calls these services itself still
records them on the consumer side.

Window Gauges reuse the current aggregation implementation. Idle-window refresh
is a separate concern tracked by #3765, not implemented or required as a code
dependency here. This change does not claim those windows refresh on scrape.

## Metric contract (#3337)

This section is the stable contract of the built-in RPC metrics: names,
labels, failure taxonomy and the RED model built on them. Dashboards and alerts
should rely only on what is listed here.

### Trace exemplars for latency histograms

Set `metrics.histogram.enabled: true` and `metrics.histogram.exemplars: true`,
or pass both `metrics.WithHistogramEnabled()` and
`metrics.WithHistogramExemplars()` to `dubbo.WithMetrics`. Both switches must be
on to enable OpenMetrics negotiation on the existing Prometheus HTTP exporter.
Enable OpenTelemetry tracing to produce exemplars, and send
`Accept: application/openmetrics-text` when scraping them.

A latency observation can carry the `trace_id` and `span_id` of its sampled
client-attempt or server Span. The default chain runs tracing before metrics,
with tracing setup and teardown outside the latency measurement. If you list
just one of these filters, the framework inserts enabled tracing before
a configured metrics filter or enabled metrics after a configured tracing
filter as needed. If you list both, their configured order takes precedence.
Place the local tracing filter before `metrics`
to capture the Span.

### Label contract (#3337)

Every RPC series carries exactly these labels, on both sides:

| Label | Value | Cardinality |
| --- | --- | --- |
| `application_name` | Local application name (consumer metrics use the local, not the remote, application) | 1 per process |
| `application_version` | Local application version | 1 per process |
| `hostname` | Local host name | 1 per process |
| `ip` | Local IP | 1 per process |
| `interface` | Service interface, e.g. `org.apache.dubbo.UserService` | Bounded by code |
| `method` | RPC method name | Bounded by code |
| `group` | Service group, empty when unset | Bounded by config |
| `version` | Service version, empty when unset | Bounded by config |
| `protocol` | RPC protocol from the service URL: `dubbo`, `tri`, ... | Bounded by config |

Deliberately **not** labels:

- **Side.** Provider and consumer already have separate metric names
  (`dubbo_provider_*`, `dubbo_consumer_*`), so a `side` label would only repeat
  the name. Use `{__name__=~"dubbo_(provider|consumer)_.*"}` to select both.
- **Error type or error code.** Each failure category has its own counter
  (below), so `requests_failed_total` keeps a single series per call identity,
  and categories can be added without changing its shape.
- **Request-scoped values.** Request IDs, remote addresses, session IDs,
  exception messages and argument values are never labels.

The series count of one RPC metric is roughly
`instances x interfaces x methods x groups x versions x protocols`.
`interface`, `method` and `protocol` are bounded by your code and
configuration. The real cardinality risks are (a) generic invocations whose
method names come from untrusted input and (b) dynamically generated groups or
versions. Avoid both, or drop those labels with Prometheus `metric_relabel_configs`.
Series are created lazily: success-only traffic never creates failure series.

### Failure taxonomy (#3337)

Provider and consumer share one taxonomy. It mirrors Java Dubbo's
`RpcException` codes as used by `MetricsSupport#getMetricsKey`.

| Category | Metric suffix | Java code | Sources in dubbo-go |
| --- | --- | --- | --- |
| timeout | `requests_timeout_total` | `TIMEOUT_EXCEPTION`, `TIMEOUT_TERMINATE` | Triple `DeadlineExceeded`; expired `context` deadline; Getty client read timeout; Dubbo `CLIENT_TIMEOUT`/`SERVER_TIMEOUT` status |
| limit | `requests_limit_total` | `LIMIT_EXCEEDED_EXCEPTION` | Triple `ResourceExhausted`; Sentinel default fallback (consumer side, see known limits) |
| service_unavailable | `requests_failed_service_unavailable_total` | `FORBIDDEN_EXCEPTION`, `NO_INVOKER_AVAILABLE_AFTER_FILTER`, `METHOD_NOT_FOUND` | Triple `Unavailable`/`PermissionDenied`; destroyed invoker; closed remoting client; provider service or method not found; Dubbo `SERVICE_NOT_FOUND` status |
| business | `requests_business_failed_total` | `BIZ_EXCEPTION` | Triple `BizError`; error returned by the provider service implementation; Dubbo exception payload with status `OK` |
| network | `requests_failed_network_total` | `NETWORK_EXCEPTION` | Confirmed transport failure (`net.OpError` at Getty write / Triple HTTP Do); no Getty session; closed Getty client |
| codec | `requests_failed_codec_total` | `SERIALIZATION_EXCEPTION` | Local serializer failure; Dubbo `SERIALIZATION_ERROR`/`BAD_REQUEST`/`BAD_RESPONSE` status |
| unknown | `requests_unknown_failed_total` | `UNKNOWN_EXCEPTION` | Everything else, e.g. a provider panic, Dubbo `SERVICE_ERROR`/`SERVER_ERROR`/`CLIENT_ERROR`, a canceled context |

Producers declare categories in one of two ways. Neither changes the error
message or breaks `errors.Is`/`errors.As`:

- **On the error.** `rpcerror.WithKind(err, kind)` or a sentinel built with
  `rpcerror.NewKindError`. Use this for framework-created errors. Sentinels keep
  their identity, so `err == base.ErrClientClosed` still works.
- **On the request observation.** `rpcerror.RecordKind(ctx, kind)` or, on the
  consumer side, `rpcerror.RecordKindFor(ctx, invocation, kind)`. Use this when
  the error must reach the caller untouched. A provider's application error is
  the main case: protocols re-encode it by its concrete type. `RecordKindFor`
  only labels the observation bound to that invocation, so a downstream call
  made with a provider's context cannot relabel the enclosing request.

`rpcerror.Kind.String()` and `ErrorType.String()` return the same stable names,
so tracing and logging can reuse the taxonomy.

Known limits:

- A dubbo-go provider replies to framework failures that happen inside the
  service call (a missing method, a panic) with status `OK` plus an exception
  payload. The wire cannot tell them apart from application exceptions, so the
  consumer counts them as business. The provider counts them precisely.
- A failure is only visible to metrics if the metrics filter wraps the filter
  that produced it. The framework places the metrics filter with the default
  filters: on the consumer, configured reference filters such as
  `sentinel-consumer` run inside it, so their rejections count as limit. On the
  provider, a configured `service.filter` such as `sentinel-provider` runs
  outside it, so the provider does not count those rejections, and a Dubbo
  consumer sees them as business (status `OK` plus an exception payload).
- Failures raised by the cluster before a provider invoker is selected, such
  as "no provider available", happen outside the per-invoker filter chain, so
  consumer metrics do not count them. Java counts them as service_unavailable.

### RED model (#3337)

| Signal | Metric (`{side}` = `provider` or `consumer`) | Notes |
| --- | --- | --- |
| Rate | `dubbo_{side}_requests_total` (Counter) | Use `rate()`; `dubbo_{side}_qps_total` is a local sliding-window gauge |
| Errors | `dubbo_{side}_requests_failed_total` and the category counters above | Categories sum to `requests_failed_total` |
| Duration | `dubbo_{side}_rt_milliseconds_{sum,last,min,max,avg}`, `dubbo_{side}_rt_milliseconds_p{50,90,95,99}`, `dubbo_{side}_rt_{avg,min,max}_milliseconds_aggregate` | Quantiles are per-instance sliding-window gauges |
| Duration (opt-in) | `dubbo_{side}_rt_milliseconds_histogram_seconds_{bucket,sum,count}` (Histogram, seconds) | Enable with `metrics.WithHistogramEnabled()` or `metrics.histogram.enabled`; aggregatable across instances |
| In flight | `dubbo_{side}_requests_processing_total` (Gauge) | |

Each `*_aggregate` gauge is the same value computed over the local sliding
window (default 120s, 10 buckets).

PromQL examples (replace `provider` with `consumer` for the caller's view):

```promql
# QPS per service
sum by (application_name, interface) (rate(dubbo_provider_requests_total[1m]))

# Success rate
sum by (interface) (rate(dubbo_provider_requests_succeed_total[5m]))
  / sum by (interface) (rate(dubbo_provider_requests_total[5m]))

# Error rate
sum by (interface) (rate(dubbo_provider_requests_failed_total[5m]))
  / sum by (interface) (rate(dubbo_provider_requests_total[5m]))

# Timeout / limit / service-unavailable rate
sum by (interface) (rate(dubbo_consumer_requests_timeout_total[5m]))
  / sum by (interface) (rate(dubbo_consumer_requests_total[5m]))
sum by (interface) (rate(dubbo_consumer_requests_limit_total[5m]))
  / sum by (interface) (rate(dubbo_consumer_requests_total[5m]))
sum by (interface) (rate(dubbo_consumer_requests_failed_service_unavailable_total[5m]))
  / sum by (interface) (rate(dubbo_consumer_requests_total[5m]))

# P99 latency across instances, with histograms enabled (seconds)
histogram_quantile(0.99, sum by (le, interface, method) (rate(dubbo_provider_rt_milliseconds_histogram_seconds_bucket[5m])))

# P99 latency without histograms: the p99 gauges cannot be averaged across
# instances; take the worst
max by (interface, method) (dubbo_provider_rt_milliseconds_p99)

# Mean latency across instances
sum by (interface) (rate(dubbo_provider_rt_milliseconds_sum[5m]))
  / sum by (interface) (rate(dubbo_provider_requests_total[5m]))

# Failures by category. rate() drops the metric name, so query each category
# counter separately: timeout, limit, failed_service_unavailable,
# business_failed, failed_network, failed_codec, unknown_failed.
# Only categories that occurred have series.
sum by (interface) (rate(dubbo_provider_requests_business_failed_total[5m]))
```

### Compatibility (#3337)

- **New `protocol` label.** It is added to every RPC series. Selectors and
  `sum by (...)` queries keep working. Queries that join on the full label set
  (`on(...)`/`ignoring(...)` listing every label) or recording rules that pin
  the exact label set need the new label. Each identity gets a separate series
  per protocol, so a service exported over both `dubbo` and `tri` now has two
  series where it used to share one.
- **Reclassified failures.** Failures that used to land in
  `requests_unknown_failed_total` now move to a precise category:
  - Dubbo timeouts and expired contexts go to timeout.
  - Dubbo business exceptions go to business.
  - Destroyed invokers, closed clients and missing service/method go to
    service_unavailable.
  - Missing Getty sessions and closed Getty clients go to network.
  - Dubbo `SERIALIZATION_ERROR`/`BAD_REQUEST`/`BAD_RESPONSE` go to codec.
  - Sentinel default-fallback rejections counted by the consumer go to limit.

  `requests_failed_total` is unchanged.
- **No renames.** No metric is renamed or removed. A few names still differ
  from the [Dubbo standard metrics](https://cn.dubbo.apache.org/zh-cn/overview/reference/metrics/standard_metrics/),
  and dubbo-go keeps its own names for compatibility:

  | Java Dubbo | dubbo-go |
  | --- | --- |
  | `requests_succeed_aggregate` | `requests_succeed_total_aggregate` |
  | `dubbo_consumer_requests_timeout_failed_aggregate` | `dubbo_consumer_requests_timeout_total_aggregate` |
  | `requests_processing` | `requests_processing_total` (Java exports both) |

  The opt-in latency histograms use the standard
  `rt_milliseconds_histogram_seconds` name and carry the same labels, including
  `protocol`.

## Callback expiry is not a business failure

A successful callback submission is not recorded as completed success.
Observation ends on a real response, context cancellation, or the supplied
request timeout (the existing default of three seconds when nonpositive).
For cancellation/expiry without a confirmed result:

- Release the observer closure, timer, context hook and Session reference.
- Synchronously record monitoring expiry to release the processing Gauge only.
- Do not invent completed totals, failure categories or response-time samples.
- Do not delete the original PendingResponse or suppress its possible late
  user callback.
- Do not revise the metrics if a real response arrives after monitoring ends.

Thus processing represents active *observations* after an unconfirmed expiry;
it is not proof that the business RPC completed. The existing no-response
callback PendingResponse retention remains unresolved. Deleting it would change
the callback contract and belongs in a separately designed completion change.
The original callback Cause behavior is preserved, even where it differs from
the decoded response error used for monitoring.

## Streaming calls

A Triple streaming call (client, server or bidi streaming) returns to the
consumer as soon as its stream is open, so the consumer records it when the
stream ends instead: the stream reaches its end, a receive or send fails, the
consumer closes the receive side, or the call's context ends, whichever comes
first. RT is the duration of the stream, the outcome and failure category come
from its final status, and `requests_processing_total` counts the stream while
it is open. A stream that fails to open is recorded at once. A stream that is
never closed and whose context never ends is never recorded, just as its HTTP/2
stream is never released. The client span of the OpenTelemetry trace filter
ends the same way.

A provider's streaming handler runs inside the filter chain until the stream
ends, so the provider already records the whole stream.

## Verification

Default focused tests (no external services):

```sh
go test ./common/rpcerror ./filter/metrics ./metrics/rpc/... ./remoting ./remoting/getty ./protocol/dubbo/... ./protocol/triple/triple_protocol
go test -race ./common/rpcerror ./filter/metrics ./metrics/rpc/... ./remoting
go test -race ./protocol/dubbo/... ./protocol/triple/triple_protocol -run 'TestResponseDecodeFailureCorrelation|TestSerializationFailureOrigin|TestLateResponseWithoutPending|TestCodecFailureOrigins|TestNetworkOriginThroughUnaryTransports'
```

The default cross-module callback test uses a deterministic transport with the
production DubboInvoker, ExchangeClient, Metrics Filter, event bus, Collector
and Prometheus registry. It verifies early/delayed failure, cancellation,
duplicate response handling and preserved user callback behavior under race.

The opt-in loopback fixture uses the production Metrics Filter, event bus,
Collector, Prometheus registry, DubboInvoker, ExchangeClient, Getty TCP client
and Dubbo/Hessian codec. A controlled peer sends real success, business-error
and malformed frames. The Network case makes a real Triple HTTP dial to a
closed local port.

```sh
go test -tags=rpc_failure_integration ./metrics/rpc/integration -count=3 -timeout=60s
go test -race -tags=rpc_failure_integration ./metrics/rpc/integration -count=1 -timeout=60s
```

The fixture covers success, request encoding failure, trusted response decoding
failure, untrusted headers, independent concurrent attempts on one Session,
a subsequent successful attempt, callback response failure, expiry followed by
a real late callback, and exported Counter/window Gauge values. Unit tests
add typed-nil/wrapping/fallback behavior, wrong Session IDs, duplicate claims,
completion races and immutable event snapshots. This is not a registry-backed
deployment, full cluster retry test, Provider wire-failure proof or Admin
browser journey.

### Known verification limits

The fixture is opt-in because Getty 1.4.10 has connection teardown races:
its `ReadTimeout`/Session state reads race with connection cleanup. These also
reproduce in the unmodified base's `TestTCPPackageHandle` and
`TestDubboProtocol_Refer`, which the root Makefile already excludes from its
race target. The opt-in fixture's unrestricted race run still exposes those
dependency races; it must not be reported as clean. No new race exclusion or
Getty dependency upgrade is introduced.

On Windows, four pre-existing Triple occupied-port tests compare platform
error text/EADDRINUSE and fail with Winsock errors. The same Triple package
passes on Linux. Use a compatible 64-bit C compiler for Windows race testing,
or Linux/WSL; an old MinGW runtime is not a reliable race toolchain.

## Remaining boundaries

- Provider request decoding and response encoding can occur outside the
  Metrics Filter; there is no general Provider wire-error accounting.
- Session-only errors without a trustworthy request ID are not attributed to
  all requests sharing the connection.
- Atomic PendingResponse claiming prevents duplicate delivery, and retired
  responses with no waiter skip Reply decoding. It does not solve the deeper
  cancellation-versus-already-started Reply decoding ownership race.
- Channel observers and other collectors still use best-effort event delivery.
  The built-in RPC recorder no longer depends on those channel deliveries.
- No new histogram, exporter deployment, Admin dashboard, wire status mapping,
  early RPC completion, callback failure contract or retry policy is included.

## Synchronous RPC recording (#3802)

The RPC recorder is installed before `Metrics.Init` returns. Core RPC updates
run in the publishing goroutine; an exhausted observer channel cannot drop a
counter, window, latency sample or processing update. `Subscribe`/`Unsubscribe`
continue to control channel observers only. Other collector event types retain
their existing asynchronous delivery policy.

A registry-owned cache binds metrics using the role and all eight existing
labels. Successful and failed results, including each failure category, bind
their counter and corresponding window only on first use. The pair is published
after both are ready. Start-only observations do not create result/latency
series, and success-only traffic does not create unused failure series.
Counters and windows are updated separately; concurrent scrapes can observe an
intermediate snapshot rather than an atomic transaction across all fields.

Registries implementing `GaugeFuncRegistry` support concurrent metric updates
and collection. Registries without it retain the legacy handlers and their
record-time window gauges under a recorder mutex. Registry panics are isolated
from the business invocation and reported once per recorder; this does not
promise complete recording after a registry failure.

Initialization remains process-wide and one-time. A start event captures the
recorder that was ready at creation; observations begun before initialization
do not acquire completion-only accounting later. Config reload, unregistering
series and switching registries at runtime are not introduced. Existing callback
observations may finish while the process is alive; the recorder owns no queue
that must be drained during shutdown. Registered series and cached handles live
for the registry lifetime, so unbounded label cardinality remains an application
concern. This is in-memory accounting, not durable delivery across process crashes.

Run `go test ./metrics/... ./filter/metrics ./common/rpcerror`, and the same
packages with `-race` on Linux. The integration subprocess tests exercise real
`Metrics.Init`, the first call after readiness, disabled RPC metrics, concurrent
calls and full observer channels. Recorder tests compare all scalar metrics with
the legacy handlers under concurrent first binding/collection and test every
existing label, result laziness, expiry and custom-registry behavior.
