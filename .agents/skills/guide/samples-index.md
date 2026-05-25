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

Each entry links to a directory under [apache/dubbo-go-samples](https://github.com/apache/dubbo-go-samples).

## Quick Start
- [helloworld](https://github.com/apache/dubbo-go-samples/tree/main/helloworld) — minimal Triple/Protobuf provider + consumer
- [direct](https://github.com/apache/dubbo-go-samples/tree/main/direct) — direct URL connection, no registry

## Protocols
- [rpc/triple](https://github.com/apache/dubbo-go-samples/tree/main/rpc/triple) — Triple variants (Protobuf, JSON, streaming, reflection, OpenAPI)
- [rpc/grpc](https://github.com/apache/dubbo-go-samples/tree/main/rpc/grpc) — gRPC interop
- [rpc/multi-protocols](https://github.com/apache/dubbo-go-samples/tree/main/rpc/multi-protocols) — multiple protocols on one server
- [http3](https://github.com/apache/dubbo-go-samples/tree/main/http3) — Triple over HTTP/3 (experimental)

## Service Discovery
- [registry/nacos](https://github.com/apache/dubbo-go-samples/tree/main/registry/nacos)
- [registry/zookeeper](https://github.com/apache/dubbo-go-samples/tree/main/registry/zookeeper)
- [registry/etcd](https://github.com/apache/dubbo-go-samples/tree/main/registry/etcd)
- [registry/polaris](https://github.com/apache/dubbo-go-samples/tree/main/registry/polaris)

## Configuration
- [config_yaml](https://github.com/apache/dubbo-go-samples/tree/main/config_yaml)
- [config_center/nacos](https://github.com/apache/dubbo-go-samples/tree/main/config_center/nacos)
- [config_center/zookeeper](https://github.com/apache/dubbo-go-samples/tree/main/config_center/zookeeper)
- [config_center/apollo](https://github.com/apache/dubbo-go-samples/tree/main/config_center/apollo)

## Filters & Resilience
- [filter/custom](https://github.com/apache/dubbo-go-samples/tree/main/filter/custom) — writing custom filters
- [filter/sentinel](https://github.com/apache/dubbo-go-samples/tree/main/filter/sentinel) — flow control
- [filter/hystrix](https://github.com/apache/dubbo-go-samples/tree/main/filter/hystrix) — circuit breaker
- [filter/token](https://github.com/apache/dubbo-go-samples/tree/main/filter/token)
- [filter/tpslimit](https://github.com/apache/dubbo-go-samples/tree/main/filter/tpslimit)
- [retry](https://github.com/apache/dubbo-go-samples/tree/main/retry)
- [timeout](https://github.com/apache/dubbo-go-samples/tree/main/timeout)
- [graceful_shutdown](https://github.com/apache/dubbo-go-samples/tree/main/graceful_shutdown)

## Traffic Management
- [router/tag](https://github.com/apache/dubbo-go-samples/tree/main/router/tag)
- [router/condition](https://github.com/apache/dubbo-go-samples/tree/main/router/condition)
- [router/script](https://github.com/apache/dubbo-go-samples/tree/main/router/script)
- [mesh](https://github.com/apache/dubbo-go-samples/tree/main/mesh) — proxyless mesh
- [apisix](https://github.com/apache/dubbo-go-samples/tree/main/apisix) — APISIX gateway

## Streaming
- [streaming](https://github.com/apache/dubbo-go-samples/tree/main/streaming)

## Observability
- [metrics/prometheus_grafana](https://github.com/apache/dubbo-go-samples/tree/main/metrics/prometheus_grafana)
- [metrics/probe](https://github.com/apache/dubbo-go-samples/tree/main/metrics/probe) — Kubernetes health probes
- [otel/tracing](https://github.com/apache/dubbo-go-samples/tree/main/otel/tracing) — Jaeger / OTLP / stdout exporters
- [logger](https://github.com/apache/dubbo-go-samples/tree/main/logger) — default, level, rolling, custom, trace integration

## Security
- [tls](https://github.com/apache/dubbo-go-samples/tree/main/tls)
- [filter/token](https://github.com/apache/dubbo-go-samples/tree/main/filter/token)

## Java Interop
- [java_interop/protobuf-triple](https://github.com/apache/dubbo-go-samples/tree/main/java_interop/protobuf-triple)
- [java_interop/non-protobuf-dubbo](https://github.com/apache/dubbo-go-samples/tree/main/java_interop/non-protobuf-dubbo)
- [java_interop/non-protobuf-triple](https://github.com/apache/dubbo-go-samples/tree/main/java_interop/non-protobuf-triple)
- [java_interop/service_discovery](https://github.com/apache/dubbo-go-samples/tree/main/java_interop/service_discovery)

## Advanced
- [generic](https://github.com/apache/dubbo-go-samples/tree/main/generic) — generic invocation
- [async](https://github.com/apache/dubbo-go-samples/tree/main/async)
- [llm](https://github.com/apache/dubbo-go-samples/tree/main/llm) — LLM integration
- [book-flight-ai-agent](https://github.com/apache/dubbo-go-samples/tree/main/book-flight-ai-agent) — AI agent
- [transcation/seata-go](https://github.com/apache/dubbo-go-samples/tree/main/transcation/seata-go) — Seata distributed transactions <!-- upstream directory is named "transcation" (typo in dubbo-go-samples); kept as-is to match actual path -->
