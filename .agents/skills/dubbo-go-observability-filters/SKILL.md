---
name: dubbo-go-observability-filters
description: >-
  Implements and reviews dubbo-go filters, metrics, tracing, logging, and related cross-cutting interception. Use when
  the user asks about Filter, Invoke, OnResponse, service.filter, reference.filter, filter chain order, TPS limiting,
  active limit, auth, token, access log, Sentinel, Seata, graceful shutdown filters, OpenTelemetry, tracing,
  Prometheus metrics, metrics probes, logger integration, or packages filter/, metrics/, otel/, logger/, and
  protocol/protocolwrapper/. Do not use for protocol wire behavior unless interception changes the transport contract.
---

# dubbo-go Observability and Filters

## Purpose

Use this skill to change or explain the filter chain and cross-cutting concerns such as metrics, tracing, logging, rate limiting, auth, and response interception.

## When to use

Use for filter registration, filter order, provider or consumer interception, `OnResponse`, TPS strategies, metrics collectors, OpenTelemetry tracing, logging integration, and probe endpoints.

Do not use for concrete wire protocol behavior unless filter behavior changes the transport contract.

## Inputs

Required:
- Filter name, metrics/tracing component, config key, or package path.
- Intended interception or telemetry behavior.

Optional:
- URL filter params, invocation attachments, context values, metric labels, span output, or log output.

If missing, inspect `filter/filter.go` and `protocol/protocolwrapper/protocol_filter_wrapper.go`.

## Workflow

1. Read `filter/filter.go` for the `Invoke` and `OnResponse` contract.
2. Read `protocol/protocolwrapper/protocol_filter_wrapper.go` for provider and consumer chain construction.
3. Inspect extension registration in `common/extension/filter.go` and any TPS extension files.
4. Read the concrete filter under `filter/`.
5. For metrics, inspect `metrics/` and the `filter/metrics/` integration.
6. For tracing, inspect `filter/otel/`, `filter/tracing/`, and `otel/`.
7. Read `references/filter-observability-flow.md` before changing filter order, metric labels, or span semantics.

## Output format

Return filter chain position, provider or consumer side, telemetry contract, changed files, and validation performed.

## Validation

- Confirm filter names are registered and selected through `service.filter` or `reference.filter`.
- Confirm filters call the next invoker exactly once unless intentionally short-circuiting.
- Confirm `OnResponse` handles both success and error results.
- Run focused tests such as `go test ./filter/... ./metrics/... ./otel/...`; use `make test` for broad interception changes.

## Edge cases

- Filter names are listed left-to-right in config, but the wrapper builds the chain from right to left so invocation runs left-to-right.
- `extension.GetFilter` returns a boolean; missing filters can currently create nil filter risks if callers ignore it.
- Metrics and tracing filters must avoid changing invocation results unless that is the feature.
- Context and invocation attachments may be protocol-specific.

## References

- Read `references/filter-observability-flow.md` for filter chain and telemetry details.
