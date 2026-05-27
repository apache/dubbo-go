# Adaptive Service P2C Healthy Provider Sample

This sample verifies the full adaptive service path across multiple direct providers:

```text
provider adaptive filter computes remaining -> response attachment returns remaining -> consumer adaptive cluster writes metrics -> P2C reads metrics -> traffic shifts toward the provider with higher remaining
```

It does not use a registry, change the proto, read response attachments in sample code, or use logs as the acceptance signal.

In this project, provider health for adaptive service is the current remaining capacity:

```text
remaining = limitation - inflight
```

## Run

Start three providers with different handler costs:

```bash
go run ./samples/adaptive_service/p2c_healthy/server \
  -id fast \
  -port 20101 \
  -stats-port 21101 \
  -work-ms 20
```

```bash
go run ./samples/adaptive_service/p2c_healthy/server \
  -id medium \
  -port 20102 \
  -stats-port 21102 \
  -work-ms 100
```

```bash
go run ./samples/adaptive_service/p2c_healthy/server \
  -id slow \
  -port 20103 \
  -stats-port 21103 \
  -work-ms 300
```

Start the client:

```bash
go run ./samples/adaptive_service/p2c_healthy/client \
  -urls 'tri://127.0.0.1:20101;tri://127.0.0.1:20102;tri://127.0.0.1:20103' \
  -stats-urls 'http://127.0.0.1:21101/stats;http://127.0.0.1:21102/stats;http://127.0.0.1:21103/stats' \
  -concurrency 200 \
  -duration 60s
```

The client writes `p2c_healthy_result.csv` by default. Use `-out ""` to disable CSV output. The `latency_avg_ms` and `latency_p95_ms` columns are calculated from successful requests in the current report interval.

The console and CSV output include both cumulative and current-window routing signals:

- `interval_providers`: successful request distribution in the last report interval.
- `providers`: cumulative successful request distribution since startup.
- `healthiest`: provider with the highest current `limiter_remaining`.
- `limiters`: latest limiter snapshot for every provider.

## Success Signals

- After warmup, every provider reports `limiter_found=true`.
- `healthiest` identifies the provider with the highest `limiter_remaining`.
- `interval_providers` is biased toward the current `healthiest` provider more often than the slower or lower-remaining providers.
- `providers` shows the cumulative hit ratio moving toward providers that repeatedly appear as `healthiest`.
- `failed` stays close to `0`.

## Useful Stats

Each provider exposes `GET /stats` with:

- `id`
- `work_ms`
- `active`
- `max_active`
- `started`
- `completed`
- `avg_handler_latency_ms`
- `limiter_found`
- `limiter_limitation`
- `limiter_remaining`
- `limiter_inflight`
