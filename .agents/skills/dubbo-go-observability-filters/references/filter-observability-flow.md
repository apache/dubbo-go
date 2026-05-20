# Filter and Observability Flow

## Filter Contract

- `filter.Filter.Invoke` receives context, next invoker, and invocation.
- `filter.Filter.OnResponse` receives the result after invocation and can observe or modify it.
- Filters should preserve result errors unless the filter intentionally converts or rejects a call.

## Chain Construction

1. Runtime exports or refers through `protocolwrapper.FILTER`.
2. Provider export reads `service.filter`; consumer refer reads `reference.filter`.
3. Filter names are parsed left-to-right from the URL param.
4. The wrapper builds from right to left so execution order is the configured left-to-right order.
5. `FilterInvoker.Invoke` calls the filter, then calls `OnResponse`.

## Registration

- Filters register through `extension.SetFilter`.
- TPS limiter, strategy, and rejected handler extensions have separate registries under `common/extension/tps_limit.go`.
- Some filters are enabled only when their package is imported.

## Observability

- Metrics filters and collectors live under `filter/metrics/` and `metrics/`.
- Probe endpoints live under `metrics/probe/`.
- OpenTelemetry and tracing behavior lives under `filter/otel/`, `filter/tracing/`, and `otel/`.
- Logger integration lives under `logger/` and is also affected by config.

## Contract Checks

- Keep labels, span names, and status classification stable unless the task explicitly changes telemetry output.
- Do not add blocking network calls to filters without timeouts.
- Avoid mutating shared invocation attachments unless the filter owns the key.
