# Adaptive Service Protect Provider Demo

This demo verifies provider-side adaptive service protection under high concurrency.

The server enables provider adaptive service and records how many requests actually enter the business handler. Rejected requests happen in the provider filter before the handler, so they do not increase `active` or `max_active`.

## Generate

```bash
protoc \
  --go_out=. --go_opt=paths=source_relative \
  --plugin=protoc-gen-go-triple=./tools/protoc-gen-go-triple/protoc-gen-go-triple \
  --go-triple_out=. \
  presee_test/adaptive_service/protect_provider/proto/protect.proto
```

## Run

Start the provider:

```bash
go run ./presee_test/adaptive_service/protect_provider/server -port 20001 -stats-port 21001 -work-ms 200
```

Start the load client:

```bash
go run ./presee_test/adaptive_service/protect_provider/client \
  -url 127.0.0.1:20001 \
  -stats-url http://127.0.0.1:21001/stats \
  -concurrency 200 \
  -duration 30s
```

The client prints one summary line per second and writes `protect_provider_result.csv` by default. Use `-out ""` to disable CSV output.

Expected signal:

- `started` keeps increasing.
- `rejected` is greater than zero.
- `failed` stays near zero.
- `server_max_active` remains clearly below `-concurrency=200`.

Example output:

```text
elapsed=10s started=840 success=610 rejected=180 failed=0 qps=61.0 reject_rate=21.4% avg=205ms p95=240ms server_active=50 server_max_active=53
```
