<!--
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

# dubbo-go-samples Index

Use this index when a user asks which example to follow. Paths refer to `https://github.com/apache/dubbo-go-samples/tree/main/<path>`.

## Quick Start

- `helloworld` - minimal Triple/Protobuf provider and consumer
- `direct` - direct URL connection without registry
- `config_yaml` - YAML-driven provider and consumer
- `context` - context propagation
- `error` - error behavior

## Protocols

- `rpc/triple/pb` - Triple with Protobuf
- `rpc/triple/pb2` - additional Triple Protobuf variant
- `rpc/triple/pb-json` - Triple JSON behavior
- `rpc/triple/instance` - `dubbo.NewInstance` style
- `rpc/triple/registry` - Triple with registry
- `rpc/triple/reflection` - reflection
- `rpc/triple/stream` - Triple streaming
- `rpc/triple/openapi` - runtime OpenAPI docs for Triple
- `rpc/triple/old_triple` - old Triple compatibility
- `rpc/grpc` - gRPC interop pattern
- `rpc/multi-protocols` - multiple protocols on one server
- `http3` - experimental Triple HTTP/3

## Service Discovery

- `registry/nacos`
- `registry/zookeeper`
- `registry/etcd`
- `registry/polaris`
- `config_center/nacos`
- `config_center/zookeeper`
- `config_center/apollo`

## Filters and Resilience

- `filter/custom` - custom filters
- `filter/token` - token filter
- `filter/tpslimit` - TPS limiting
- `filter/sentinel` - Sentinel flow control
- `filter/hystrix` - Hystrix-style circuit breaker
- `filter/polaris/limit` - Polaris limit integration
- `retry` - retry behavior
- `timeout` - timeout behavior
- `graceful_shutdown` - graceful shutdown behavior

## Traffic Management

- `router/tag` - tag routing
- `router/condition` - condition routing
- `router/script` - script routing
- `router/static_config/tag` - static tag router config
- `router/static_config/condition` - static condition router config
- `router/polaris` - Polaris router integration
- `mesh` - proxyless mesh scenario
- `apisix` - APISIX gateway integration

## Streaming and Async

- `streaming`
- `async`

## Observability and Operations

- `metrics/prometheus_grafana`
- `metrics/probe`
- `otel/tracing/jaeger`
- `otel/tracing/otlp_http_exporter`
- `otel/tracing/stdout`
- `logger/default`
- `logger/level`
- `logger/rolling`
- `logger/custom`
- `logger/trace-integration`
- `healthcheck`

## Security

- `tls`
- `filter/token`

## Java Interop

- `java_interop/protobuf-triple`
- `java_interop/non-protobuf-dubbo`
- `java_interop/non-protobuf-triple`
- `java_interop/service_discovery`
- `helloworld` - includes Go and Java sides
- `direct` - includes Go and Java direct mode
- `http3` - includes Java HTTP/3 sample

## Advanced

- `generic` - generic invocation
- `llm` - LLM-related sample
- `book-flight-ai-agent` - AI agent sample
- `game`
- `online_boutique`
- `transcation/seata-go` - Seata transaction sample; upstream directory name is intentionally misspelled `transcation`
