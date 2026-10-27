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
6. The Collector consumes an immutable event and increments total failure and
   exactly one category. No transport or codec increments counters directly.

A successful result remains successful even if sidecar evidence exists.
Observation freezing is protected by a mutex. Response, cancellation and
monitoring timeout compete for one monitoring terminal state. Evidence arriving
after that state cannot change an already published event. Retries have
independent observations: a failed local attempt followed by a successful
attempt is not one successful metric event and is not one permanent failure.

## Classification

Existing DeadlineExceeded, ResourceExhausted, PermissionDenied and BizError
priority is retained. An unmarked Unavailable remains ServiceUnavailable.
Other failures may use confirmed local Network or Codec evidence; plain
Internal/InvalidArgument, error text, or a Session close alone do not prove
either origin. Conflicting evidence remains Unknown.

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

These use the same application/instance/interface/method/group/version labels
as the existing RPC metrics. There are no request-ID, Session, exception text
or other high-cardinality labels. Total failure is incremented once alongside
one category; Network/Codec failures do not also increment Unknown.

Window Gauges reuse the current aggregation implementation. Idle-window refresh
is a separate concern tracked by #3765, not implemented or required as a code
dependency here. This change does not claim those windows refresh on scrape.

## Callback expiry is not a business failure

A successful callback submission is not recorded as completed success.
Observation ends on a real response, context cancellation, or the supplied
request timeout (the existing default of three seconds when nonpositive).
For cancellation/expiry without a confirmed result:

- Release the observer closure, timer, context hook and Session reference.
- Publish a monitoring-expiry event to release the processing Gauge only.
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
- Triple later streaming messages do not share the unary Filter completion
  boundary and are not claimed as full streaming coverage.
- Session-only errors without a trustworthy request ID are not attributed to
  all requests sharing the connection.
- Atomic PendingResponse claiming prevents duplicate delivery, and retired
  responses with no waiter skip Reply decoding. It does not solve the deeper
  cancellation-versus-already-started Reply decoding ownership race.
- The existing metrics event bus can drop events when full. Exactly-once
  publication is not guaranteed delivery; processing can still drift under
  event loss.
- No new histogram, exporter deployment, Admin dashboard, wire status mapping,
  early RPC completion, callback failure contract or retry policy is included.
