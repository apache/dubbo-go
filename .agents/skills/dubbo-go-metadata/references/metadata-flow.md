# Metadata Flow

## Core State

- `metadata/metadata.go` stores `registryMetadataInfo` keyed by registry ID.
- `AddService` records exported provider URLs.
- `AddSubscribeURL` records consumer subscription URLs.
- `GetMetadataService` returns the metadata service delegate backed by the same map.

## MetadataInfo

- `metadata/info/` owns application metadata, exported services, subscribed URLs, service info, and serialization behavior.
- Provider export should add service URLs before application-level service discovery registers instances.
- Consumer refer paths can add subscribed URLs for metadata reporting.

## Metadata Reports

- Interface is in `metadata/report/report.go`.
- Factories register through `common/extension/metadata_report_factory.go`.
- Backends include Nacos, Zookeeper, and Etcd.
- Remote metadata storage depends on a report instance created from registry or metadata report config.

## Metadata Service

- `metadata/metadata_service.go` exports Dubbo and Triple metadata services.
- V1 and V2 generated service info lives in the same file and under `metadata/triple_api/`.
- The service delegates reads to `DefaultMetadataService`, which reads `MetadataInfo` by revision or current data.

## Service Name Mapping

- Mapping interfaces live under `metadata/mapping/`.
- Application-level discovery uses mapping to translate interface service names into provider application names.
- Mapping storage may be provided by metadata reports.

## Contract Checks

- Keep revision changes synchronized between `MetadataInfo`, service instance metadata, and report storage.
- Do not publish nil or incomplete metadata before service discovery registration.
- Preserve compatibility for Java class names and Dubbo metadata service method names.
