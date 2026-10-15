# Admin condition-route fixtures

The service/application rules use Admin's existing `ConditionRoute` API. The
instance fixtures are produced by `CreateConditionRule` and
`UpdateInstanceTrafficStatus`; restoring an instance keeps `=>env=stable`.

- Admin base: `b9ee1b779fd7e87d0097edb5645e9c9176a917a7`.
- Writer used: `71c70cf9cff3302e687cf0e482017277e9ff8ed0`
  (`meshresource.MarshalRuleSpec`, called by the Nacos and ZooKeeper governors).
- `legacy-enabled-omitted.yaml` is the unmodified old writer's disabled-service
  payload, with only the Apache comment header added. Its missing `enabled` cannot express a disabled rule. It is kept to
  check that omitted flags do not crash the Go router.

To regenerate, check out the writer revision in a separate Admin checkout, then
run from this directory:

```sh
./regenerate.sh /path/to/dubbo-admin
```

Apache comment headers are added to the generated YAML; the writer payload is unchanged.

The normal test suite is self-contained:

```sh
go test ./cluster/router/condition/... -count=1
go test -race ./cluster/router/condition -count=1
```

For real configuration-center and Triple RPC checks, run the services below on
unused ports, then set their addresses. An explicitly configured address makes an
unavailable service fail the test.

```sh
docker compose -f compose.yaml up -d --wait
DUBBO_GO_CONDITION_NACOS=127.0.0.1:18848 \
DUBBO_GO_CONDITION_ZK=127.0.0.1:21811 \
go test ./cluster/router/condition -run '^TestAdminConditionConfigCenterRPC$' -v -count=1
```

The test starts real Triple providers on random local ports. A uses `127.0.0.1`;
B uses a local non-loopback IPv4 address. Set `DUBBO_GO_CONDITION_PROVIDER_B` to
choose B explicitly. Both addresses must reach this machine. Admin disables an
IP, so A and B cannot share a host with different ports. C uses `env=canary` to
prove that restoring A retains the unrelated `env=stable` condition.

The raw writes use Admin's transport contract: Nacos data ID = resource name and
`group=dubbo`; ZooKeeper path = `/dubbo/config/<resource-name>`. The Go ZooKeeper
client is configured with `dubbo.config-center.root-path=/dubbo` and
`config-center.namespace=config`. Its defaults read
`/dubbo/config/dubbo/<key>` and do not match this Admin revision.

Full lifecycle checks also need the Nacos SDK deletion fix (`v2.3.5`) and GOST's
`StateSyncConnected` watch-event dispatch fix. The base Go revision uses SDK
`v2.2.5` and GOST `v1.14.5`: native Nacos deletion and ZooKeeper updates fail with
those versions. The tests keep these failures visible rather than calling
`Process` directly in the RPC run.

The test checks a rule present before subscription, updates, rule disable/enable,
forced empty results and deletion. The application case also checks disabling A
and restoring it. A separate consumer-application check loads a rule before
construction while the provider application has no rule. Each selected invoker makes a real RPC and must return its
provider identity. An empty forced result has no invoker to call.

The fixture IP `127.0.0.2` is replaced with B's reachable IP for the RPC run.
The wire format and configuration key are unchanged. No interface-level
registration or discovery mode is enabled by these tests.
