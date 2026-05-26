# Adaptive Service RTT Shrink Sample

This sample verifies that provider-side adaptive service shrinks the method limiter after handler RTT increases.

The verification does not depend on log files or response attachments. The server exposes `GET /stats`, which reads the provider filter middleware limiter through `adaptivesvc.GetMethodLimiterSnapshot` and returns the current `limiter_limitation`.

## Run

Start the provider:

```bash
go run ./samples/adaptive_service/rtt_shrink/server \
  -port 20002 \
  -stats-port 21002 \
  -stages fast:20:30s,medium:100:20s,slow:500:30s
```

Start the client:

```bash
go run ./samples/adaptive_service/rtt_shrink/client \
  -url 127.0.0.1:20002 \
  -stats-url http://127.0.0.1:21002/stats \
  -concurrency 200 \
  -duration 60s
```

The client writes `rtt_shrink_result.csv` by default. Use `-out ""` to disable CSV output. The `latency_avg_ms` and `latency_p95_ms` columns are calculated from successful requests in the current report interval, not from all successful requests since startup.

For backward-compatible two-stage runs, the server also supports:

```bash
go run ./samples/adaptive_service/rtt_shrink/server \
  -port 20002 \
  -stats-port 21002 \
  -work-ms 50 \
  -slow-after 20s \
  -slow-work-ms 300
```

## Success Signals

- `phase` changes across the configured stages.
- `latency_avg_ms` and `latency_p95_ms` increase in higher-latency stages.
- `limiter_found` is `true`.
- During the slow phase, `limiter_limitation` drops below the fast-phase peak or stable value.
- `rejected` increases because adaptive service rejects requests at the provider-side limiter.
- `failed` stays close to `0`.
- `server_active` is controlled by the limiter and does not grow without bound with client concurrency.

## Useful Stats

`GET http://127.0.0.1:21002/stats` returns:

- `phase`
- `work_ms`
- `active`
- `max_active`
- `started`
- `completed`
- `avg_handler_latency_ms`
- `limiter_found`
- `limiter_inflight`
- `limiter_remaining`
- `limiter_limitation`
