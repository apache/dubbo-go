---
name: dubbo-go-metadata
description: >-
  Implements and reviews dubbo-go metadata changes. Use when the user asks about MetadataInfo, metadata service V1 or
  V2, metadata reports, app metadata publishing or fetching, service name mapping, metadata revision, subscribed or
  exported service URLs, remote metadata storage, or packages metadata/, metadata/info/, metadata/report/, and
  metadata/mapping/. Do not use for registry subscription mechanics unless metadata controls the behavior.
---

# dubbo-go Metadata

## Purpose

Use this skill to change or explain how dubbo-go collects exported and subscribed URLs, calculates and publishes metadata, exposes metadata services, and maps service names to applications.

## When to use

Use for `MetadataInfo`, metadata service export, metadata reports, metadata mapping, revisions, remote metadata storage, and application-level service discovery metadata interactions.

Do not use for concrete registry notification mechanics unless metadata controls the contract.

## Inputs

Required:
- Metadata type, registry ID, metadata report backend, service name mapping, revision, or metadata service behavior.
- Intended metadata behavior or observed mismatch.

Optional:
- Provider URL, consumer URL, service instance metadata, app name, revision string, or metadata report payload.

If missing, start at `metadata/metadata.go` and follow into `metadata/info/`, `metadata/report/`, or `metadata/mapping/`.

## Workflow

1. Read `metadata/metadata.go` for global metadata storage and add/get helpers.
2. Inspect `metadata/info/` for `MetadataInfo` structure, URL addition, and revision-related behavior.
3. For metadata service behavior, read `metadata/metadata_service.go` and generated files under `metadata/triple_api/`.
4. For reports, inspect `metadata/report/report.go`, the backend package, and extension registration.
5. For service name mapping, inspect `metadata/mapping/` and the registry service discovery caller.
6. Verify how registry IDs connect metadata to registry URLs.
7. Read `references/metadata-flow.md` before changing revision, report, or service mapping behavior.

## Output format

Return metadata producer and consumer, revision or report contract, affected registry interaction, changed files, and validation performed.

## Validation

- Confirm exported and subscribed URLs are stored under the correct registry ID.
- Confirm remote metadata storage publishes before service discovery depends on it.
- Confirm metadata service V1 and V2 compatibility when changing service methods or response models.
- Run focused tests such as `go test ./metadata/...`; include `go test ./registry/servicediscovery/...` when service discovery depends on metadata.

## Edge cases

- Metadata globals are keyed by registry ID; missing registry IDs can merge unrelated data.
- Metadata service V1 and V2 expose different wire shapes but share the same delegate.
- Service name mapping is used by application-level discovery and may be backed by metadata reports.
- Remote metadata storage requires a metadata report instance.

## References

- Read `references/metadata-flow.md` for metadata collection, report, mapping, and service export details.
