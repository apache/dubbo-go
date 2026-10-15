# Admin tag-rule fixtures

The `enabled.yaml`, `updated.yaml`, `force.yaml`, and
`legacy-disabled-omitted.yaml` payloads were read back from ZooKeeper after
`RuleGovernor.CreateRule` / `UpdateRule` in dubbo-admin
`b9ee1b779fd7e87d0097edb5645e9c9176a917a7` wrote them. `fixed-disabled.yaml`
was captured through the same writer at
`71c70cf9cff3302e687cf0e482017277e9ff8ed0`.

The old writer omits `enabled: false`. That payload still enables the Go rule,
because an omitted `enabled` defaults to true. The fixed writer emits false
explicitly; the tests check both payloads without changing the SDK default.
Nacos uses the same serialized spec as ZooKeeper.

To capture the fixtures again, run this from a checkout of the specified Admin
revision with a disposable ZooKeeper server. Use absolute paths for the generator
and output directory:

```sh
GOTOOLCHAIN=go1.25.0 go run /path/to/dubbo-go/cluster/router/tag/testdata/admin/generate.go \
  -zookeeper zookeeper://127.0.0.1:2181 -out /tmp/tag-fixtures
```

The generator writes `/dubbo/config/tag-compat-provider.tag-router`, captures
four payloads, then deletes that node. `disabled.yaml` corresponds to the legacy
or fixed disabled fixture, depending on the Admin revision. The local Admin
resource cache is deliberately unused; the captured bytes come from the real
ZooKeeper writer.

## Tests

Self-contained regression tests need no config center:

```sh
go test -count=1 ./cluster/router/tag
```

The external test starts two real Triple providers, then loads and watches their
provider application's rule. The consumer has a different application name.
It invokes the selected providers and compares their returned addresses for
both cached and uncached routing. It checks updates, disabling, fallback,
force, and actual config deletion.

```sh
DUBBO_TAG_NACOS=127.0.0.1:8848 \
DUBBO_TAG_ZOOKEEPER=127.0.0.1:2181 \
go test -tags adminintegration -run TestAdminTagConfigCenterRPC -count=1 -v ./cluster/router/tag
```

An unset endpoint skips that backend. Each run uses a unique application key
and removes it afterwards. Use disposable services; the Nacos endpoint must
accept unauthenticated configuration requests.

Admin's Nacos location is group `dubbo`, data ID `<provider-app>.tag-router`.
Go must set `config-center.group=dubbo`; Nacos's `DEFAULT_GROUP` is different.
Admin's ZooKeeper location is `/dubbo/config/<provider-app>.tag-router`.
The test sets Go's `dubbo.config-center.root-path=/dubbo` and
`config-center.namespace=config` to read and watch that exact path. Go's default
root and namespace would add another `/dubbo` component.

These tests do not restore interface-level discovery. They start from two
provider invokers, as supplied to the tag router by application discovery, and
exercise the config-center and RPC boundary directly.
