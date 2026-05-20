# Config Flow

## Entry Points

- `loader.go`: top-level `dubbo.Load`, file watch, hot reload, and `InstanceOptions`.
- `config/config_loader.go`: package-level `config.Load`, `RootConfig` storage, and legacy config accessors.
- `dubbo.go`: maps `InstanceOptions` into `client.ClientOption` and `server.ServerOption`.
- `global/`: config models used by public client/server options.
- `config/`: builders, compatibility models, config center startup, and legacy `RootConfig` initialization.

## Loader Flow

Top-level `dubbo.Load`:
1. Build `loaderConf`.
2. Resolve config with koanf from file, raw bytes, or direct options.
3. Unmarshal into `InstanceOptions`.
4. Run `InstanceOptions.init`.
5. Create an `Instance` and start consumer/provider loading.
6. Start the file watcher once and register shutdown cleanup.

Package `config.Load`:
1. Build `LoaderConf`.
2. Resolve into `RootConfig` when one is not provided.
3. Run `RootConfig.Init`.
4. Store the initialized root config through the atomic pointer.

## Config Center

- Config center setup is coordinated by `config/config_center_config.go`.
- Implementations live under `config_center/{apollo,file,nacos,zookeeper}`.
- Factories register through `common/extension/config_center_factory.go`.
- Parser behavior is under `config_center/parser/`.

## Runtime Injection

- `Instance.NewClient` clones consumer, application, registry, shutdown, metrics, otel, TLS, protocol, and router config into `client` options.
- `Instance.NewServer` clones application, registry, protocol, shutdown, metrics, otel, and TLS config into `server` options.
- Reference and service options ultimately build `common.URL` values. Use params for transportable Dubbo URL values and attributes for in-process objects.

## Hot Reload

- File watching is in `loader.go`.
- `hotUpdateConfig` compares old and new koanf trees with `safeChanged`.
- Keep hot reload allowlists conservative because changing immutable runtime fields after export or refer can leave stale protocols, registries, or metadata.
