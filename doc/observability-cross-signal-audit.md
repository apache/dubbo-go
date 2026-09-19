# Observability cross-signal audit

Parent: [#3562](https://github.com/apache/dubbo-go/issues/3562) Workstream 1.
Issue: [#3720](https://github.com/apache/dubbo-go/issues/3720).

This document records **observed** Metrics, Trace, Log, and Metadata representations of service, method, protocol, side, and error. It does not change exported behavior.

**Audited revision:** `develop@9af40a2b835b5176be453b5f3936b1fc53a7ead4` (2026-09-18, `#3732`).

Statements about current behavior cite source or tests. Proposed or unmerged work is labeled as such. Unresolved mappings are listed as questions.

## 1. Cross-signal inventory

`N/A` means the signal has no field for that dimension today. `absent` means the dimension exists on another signal but not this one.

| Dimension | Metrics | Trace (OTel RPC filter) | Log (Zap/Logrus CtxLogger) | Metadata (`ServiceInfo`) |
| --- | --- | --- | --- | --- |
| Application | label `application_name` from URL `application` | resource `service.name` from `Application.Name` | absent | `MetadataInfo.App` |
| Application version | label `application_version` from URL `app.version` | resource `service.version` from `Application.Version` | absent | absent (URL param `version` is **service** version) |
| Service / interface | label `interface` = `url.Service()` | attribute `rpc.service` = `url.Service()` | access-log map key `interface` from attachments; CtxLogger has no service field | `ServiceInfo.Name`; map key is match key (`ServiceKey` + protocol) |
| Method | label `method` = `invocation.MethodName()` | attribute `rpc.method` = `invocation.MethodName()`; **span name** = `invocation.ActualMethodName()` | access-log map key `method` from attachments; CtxLogger has no method field | method names live in `ServiceInfo.Params` (`methods` key), not a first-class field |
| Group | label `group` = `url.Group()` | attribute `dubbo.group` when group is non-empty | access-log map key `group` from attachments | `ServiceInfo.Group` |
| Version (service) | label `version` = URL `version` | attribute `dubbo.version` when version is non-empty | access-log map key `version` from attachments | `ServiceInfo.Version` |
| Protocol | absent on RPC metrics | absent on OTel RPC spans | absent on CtxLogger | `ServiceInfo.Protocol` |
| Side | encoded in **metric name** (`dubbo_provider_*` / `dubbo_consumer_*`), not a `side` label | `SpanKind` `SERVER` / `CLIENT` | absent | N/A (metadata is instance/service catalog) |
| RPC system | N/A | `rpc.system` = `apache_dubbo` | N/A | N/A |
| Instance | labels `ip`, `hostname` | absent on RPC span (resource has no host IP attribute) | access-log `local-addr` / `remote-addr` | `ServiceInfo.Port` / `Path`; instance host is on the registry instance, not `ServiceInfo` |
| Error class | separate counter names (timeout / limit / unavailable / business / unknown); **no** `error` / `error_type` label | span status `Error` + `SetStatus` description = `err.Error()`; no `error_type` attribute | CtxLogger `Error*` methods; no structured `error_type`. Trace IDs only on context logs | mapping/fetch metrics use `result` / success-fail suffixes, not RPC error taxonomy |
| Trace correlation | N/A (no exemplars) | span context | CtxLogger fields `trace_id`, `span_id`, `trace_flags` when span context is valid | N/A |

### Composite keys (not a Prometheus / OTel RPC attribute)

`url.ServiceKey()` formats `group/interface:version` (`common/url.go` `ServiceKey`). Version `0.0.0` is omitted from the key. RPC metric `version` and trace `dubbo.version` still emit the raw URL param, so `0.0.0` can appear there while being absent from `ServiceKey()`. Registration, routing, and `MetadataInfo` URL maps still use `ServiceKey()`. It is **not** written to RPC metric `interface` or OTel `rpc.service` on this revision.

Legacy OpenTracing filter `filter/tracing/filter.go` still names spans `ServiceKey() + "#" + MethodName()`. That filter is separate from OTel (`otelServerTrace` / `otelClientTrace`).

## 2. Per-signal detail

### 2.1 Metrics (RPC)

**Location:** `metrics/rpc/util.go` `buildLabels`; `metrics/rpc/collector.go`; `metrics/rpc/metric_set.go`; `metrics/rpc/error_classifier.go`.

**Labels (all RPC time series):** `application_name`, `application_version`, `hostname`, `ip`, `interface`, `method`, `group`, `version` (`common/constant/metric.go`).

**Side:** `getRole` reads URL `registry.role` (`3` provider, `0` consumer) and selects the `dubbo_provider_*` or `dubbo_consumer_*` metric family. There is no `side` label.

**Error:** `classifyError` maps Triple/gRPC `CodeOf(err)` to `ErrorType`, then increments a dedicated counter. It does **not** attach `error` / `error_type` labels. `TagErrorCode` (`"error"`) is declared in `common/constant/metric.go` and unused.

| ErrorType | Metric suffix (provider example) | Triple codes |
| --- | --- | --- |
| Timeout | `dubbo_provider_requests_timeout_total` | `CodeDeadlineExceeded` |
| Limit | `dubbo_provider_requests_limit_total` | `CodeResourceExhausted` |
| Service unavailable | `dubbo_provider_requests_failed_service_unavailable_total` | `CodeUnavailable`, `CodePermissionDenied` |
| Business | `dubbo_provider_requests_business_failed_total` | `CodeBizError` |
| Unknown (default) | `dubbo_provider_requests_unknown_failed_total` | everything else, including Dubbo-protocol errors |
| NetworkFailure / Codec | reserved constants; **not produced** | documented TODO in `error_classifier.go` |

**Protocol:** not a label. Dubbo-protocol errors are unclassified (`TODO: Support dubbo protocol error classification`).

**Tests:** `metrics/rpc/util_test.go` (`interface` = `url.Service()`); `metrics/rpc/error_classifier_test.go`; `metrics/rpc/event_test.go`; `filter/metrics/filter_test.go`.

**Enablement:** metrics filter is **not** in `DefaultServiceFilters` / `DefaultReferenceFilters`. `server/action.go` / `client/action.go` append `metrics` only when `Metrics.Enable` is true.

### 2.2 Trace (OpenTelemetry RPC filter)

**Location:** `filter/otel/trace/filter.go` `rpcSpanAttributes`; resource in `otel/trace/exporter.go` `NewExporter`; wiring in `instance_options_init.go` `initGlobalOtel`.

**Span name:** `invocation.ActualMethodName()` (generic `$invoke` unwraps the real method; otherwise same as `MethodName()`). Not `interface/method`.

**SpanKind:** provider `SERVER`, consumer `CLIENT`.

**Attributes (merged):**

| Key | Source | When emitted |
| --- | --- | --- |
| `rpc.system` | `semconv.RPCSystemApacheDubbo` | always |
| `rpc.service` | `url.Service()` (interface param, Path fallback) | always |
| `rpc.method` | `invocation.MethodName()` | always |
| `dubbo.group` | `url.Group()` | non-empty |
| `dubbo.version` | `url.Version()` | non-empty |

Absent on the RPC span: protocol, side string, IP, hostname, error type/code.

**Resource:** `service.namespace` = application organization; `service.name` = application name; `service.version` = application version.

**Errors:** `span.SetStatus(codes.Error, res.Error().Error())` — free-form text, high cardinality. No `RecordError` on the success/failure path in this filter (CtxLogger can `RecordError` on `CtxError*` when configured).

**Propagation:** W3C TraceContext+Baggage or B3, injected/extracted via Dubbo attachments (`filter/otel/trace/attachment.go`).

**Tests:** `filter/otel/trace/filter_test.go` (`Test_otelServerFilter_Invoke_RPCAttributes`, client counterpart, `Test_rpcSpanAttributes_PreserveServiceKey`); `protocol/triple/triple_invoker_test.go` (traceparent isolation).

**Enablement:** `otelServerTrace` / `otelClientTrace` appended when `Otel.TracingConfig.Enable` is true.

**Unmerged proposal (not observed):** draft [#3551](https://github.com/apache/dubbo-go/pull/3551) would rename spans to `dubbo.consumer <service>/<method>` / `dubbo.provider <service>/<method>` and add `dubbo.side`, `dubbo.protocol`, `server.address`, `server.port`. It still used `ServiceKey()` for `rpc.service` at last update. [#3724](https://github.com/apache/dubbo-go/pull/3724) already merged `url.Service()` + `dubbo.group` / `dubbo.version` on `develop`. Treat #3551 as an open proposal, not current behavior.

### 2.3 Log

**CtxLogger (Zap / Logrus):** `logger/core/zap/ctx_logger.go`, `logger/core/logrus/ctx_logger.go`; fields from `logger/trace_extractor.go`.

When `trace.SpanContextFromContext` is valid:

| Field | Source |
| --- | --- |
| `trace_id` | `spanCtx.TraceID().String()` |
| `span_id` | `spanCtx.SpanID().String()` |
| `trace_flags` | `spanCtx.TraceFlags().String()` (sampled → `"01"`) |

Invalid/missing span: those keys are omitted (not empty strings). No `service`, `method`, `group`, `version`, `error_type`.

**Tests on this revision:** `logger/core/zap/ctx_logger_test.go`, `logger/core/logrus/ctx_logger_test.go`, `logger/trace_extractor_test.go`. Dynamic log-level + exact-ID coverage is [#3721](https://github.com/apache/dubbo-go/issues/3721) / [#3744](https://github.com/apache/dubbo-go/pull/3744) and is **not** in `develop@9af40a2b`.

**Access log filter:** `filter/accesslog/filter.go` `buildAccessLogData`. Keys from invocation attachments: `interface` (fallback `path`), `method`, `version`, `group`, `timestamp`, `local-addr`, `remote-addr`, plus `types` / `arguments`. Not JSON CtxLogger fields; not trace-correlated unless the caller put IDs in attachments.

**OpenTracing log baggage:** `filter/tracing` logs `ErrorMsg` / `Success` on the span, not the application logger.

### 2.4 Metadata

**Catalog (`ServiceInfo`):** `metadata/info/metadata_info.go`. Fields: `name`, `group`, `version`, `protocol`, `port`, `path`, `params`. `ServiceKey` / `MatchKey` are computed, not Hessian-exported.

**Metrics:** `metrics/metadata/metric_set.go`. Application-level tags plus, where used, `interface` (store-provider), `provider_app`, `result`, `source`, `storage_type` (fetch). Mapping metrics are success/fail counters, not RPC RED labels.

**Tests:** `metadata/info/metadata_info_test.go`; `metrics/metadata/collector_test.go`.

## 3. Cardinality

**Default Prometheus RPC labels (bounded, still multiply series):** application name/version, hostname, IP, interface, method, group, version. Provider vs consumer **doubles** families via metric names rather than a `side` label.

**Must not become default Prometheus labels:** `trace_id`, `span_id`, `trace_flags`, full exception strings, payloads, attachment values, revisions, raw URLs. Trace span status description currently **is** the raw error string (high cardinality in backends that index status description).

**Opt-in / other:** metadata fetch labels `provider_app`, `result`, `source`, `storage_type`; config-center `key`, `group`, `config_center`, `change_type`.

## 4. Compatibility constraints

- **Do not rename** existing `dubbo_provider_*` / `dubbo_consumer_*` metric names or the RPC label set without a deprecation path (`#3337`).
- **`url.ServiceKey()`** remains the registration/routing key. Do not change it to “fix” telemetry.
- **`rpc.service` on `develop`** is the interface FQDN (`url.Service()`), aligned with metric `interface` (`#3724`). Downstream that parsed `group/interface:version` from OTel `rpc.service` must use `dubbo.group` / `dubbo.version` or metric labels instead.
- **Span name** is still the method (or generic actual method). Changing it is a Jaeger/Tempo UX change, not required for Admin Trace→Metric mapping.
- **CtxLogger field names** stay `trace_id` / `span_id` / `trace_flags` (not Java MDC `traceId` / `spanId`).
- **Metrics and OTel filters are opt-in.** Defaults in `common/constant/default.go` do not include them.
- **`classifyError`** is Triple/gRPC-only; Dubbo-protocol failures increment `*_unknown_failed_total`.
- **`TagErrorCode`** is unused; introducing an `error` label would change cardinality and is out of scope here.

## 5. Follow-up matrix

| Observed gap | Files | Related issue/PR | Public owner (if recorded) | Validation |
| --- | --- | --- | --- | --- |
| RPC `rpc.service` vs metric `interface` (was `ServiceKey`) | `filter/otel/trace/filter.go`, `metrics/rpc/util.go` | Merged `#3724` on `develop`. Draft `#3551` still proposes `ServiceKey()` + span rename — do not treat as current. Admin mapping: `#3562` comment, `#3723` | `#3724` AlicY; `#3551` jiaming2li (draft) | Provider/Consumer span tests with `group`/`version`; metric `interface` == `url.Service()` |
| Span name is method-only | `filter/otel/trace/filter.go` | `#3551` (proposal); Admin P0 marked not required | `#3551` draft | Compare Jaeger span list vs `rpc.service`+`rpc.method` |
| No protocol on RPC metrics/trace | `metrics/rpc/util.go`, `filter/otel/trace/filter.go` | `#3337`, `#3338` | unassigned here | Call Triple and Dubbo; confirm neither series/span carries protocol |
| No `side` label (name prefix instead) | `metrics/rpc/metric_set.go`, `metrics/rpc/util.go` `getRole` | `#3337` (cardinality of adding `side`) | `#3337` | Count series with/without a hypothetical `side` label |
| Error class is metric **name**, not a shared `error_type` across Trace/Log | `metrics/rpc/error_classifier.go`, `filter/otel/trace/filter.go`, CtxLogger | `#3562` Workstream 2; `#3337` | unassigned | Timeout vs business error: metrics counters vs span status text vs logs |
| Network/codec `ErrorType` unused; Dubbo protocol unclassified | `metrics/rpc/error_classifier.go` | `#3337` | TODO in source | Force hessian decode failure / connection refuse; expect `*_unknown_failed_total` |
| `TagErrorCode` unused | `common/constant/metric.go` | `#3337` | — | `rg TagErrorCode` remains definition-only |
| Generic invoke: span name uses `ActualMethodName`, `rpc.method` uses `MethodName` (`$invoke`) | `filter/otel/trace/filter.go`, `protocol/invocation/rpcinvocation.go` | `#3338` | — | Generic `$invoke` call; compare span name vs `rpc.method` |
| `ServiceKey()` drops version `0.0.0`; metric/trace `version` keep the raw param | `common/url.go` `ServiceKey`, `metrics/rpc/util.go`, `filter/otel/trace/filter.go` | — | — | URL `version=0.0.0`; compare `ServiceKey()`, metric label, `dubbo.version` |
| CtxLogger has no service/method fields | `logger/core/zap/ctx_logger.go`, `logger/core/logrus/ctx_logger.go` | `#3562` Workstream 2; `#3721` (level+trace fields, not merged on this pin) | `#3721` AsperforMias | JSON log line contains trace fields only (`ctx_logger_test.go` on this revision) |
| Access log vs CtxLogger are different schemas | `filter/accesslog/filter.go` | `#3701` (shutdown), not a field-contract issue | — | Enable `accesslog` path; confirm no `trace_id` unless attached |
| Metadata diagnostics vs RPC RED | `metrics/metadata/metric_set.go` | `#3356`, `#3463` / mapping PRs | `#3356` | Mapping listen/get/register counters; no `method` label |

## 6. Open questions

These are **not** resolved by this audit:

1. Should RPC metrics grow a low-cardinality `protocol` or `side` label, or stay name-encoded for side?
2. Should Trace/Log reuse `classifyError` (or a shared taxonomy) instead of raw `err.Error()` on span status?
3. Should generic calls set `rpc.method` to `ActualMethodName()` so it matches the span name?
4. Is `#3551` span-name work still wanted after `#3724`, or should the draft be closed as superseded?
5. Should access logs gain `trace_id` without putting it on Prometheus?

## 7. What this document is not

No metric renames, no label cardinality changes, no shared error-classification package, no span topology changes, no dashboards. Metrics remain `#3337`; tracing `#3338`; metadata `#3356` / `#3499`; default filter wiring `#3568`.
