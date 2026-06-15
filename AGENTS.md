<!--
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements. See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License. You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

# Agent Guidance

This file gives AI coding agents the repo-level context needed to make small,
reviewable changes in Apache Dubbo-go. More specific `AGENTS.md` files may be
added in subdirectories when a package needs narrower rules.

## Project Scope

- Work from the latest `develop` branch unless a maintainer asks for another
  base branch.
- Keep pull requests focused on one issue or one tightly related behavior.
- Every pull request should reference its tracking issue, following
  `CONTRIBUTING.md`.
- Avoid changing public APIs, wire formats, configuration keys, or generated
  files unless the issue explicitly calls for that change.

## Code Style

- Use Go 1.24 or later.
- Run `gofmt` on changed Go files.
- Keep the existing grouped import style: standard library, third-party
  packages, then Dubbo-go packages.
- Prefer existing helpers and package conventions over adding new abstractions.
- Add comments only when they clarify non-obvious behavior or compatibility
  constraints.

## Testing

- Run the narrowest relevant `go test` package set for the changed files.
- For shared packages, also run dependent package tests when the dependency
  surface is obvious.
- Run `git diff --check` before committing.
- For race-sensitive changes, run the relevant race target or
  `go test -race` package set when practical.

## Common Areas

- `common/extension`: keep registries concurrency-safe and avoid exposing
  mutable internal maps or slices.
- `common.URL`: avoid returning or mutating shared internal state without clear
  locking or copy semantics.
- `config`: preserve deprecated compatibility paths unless the tracking issue
  explicitly removes them.
- `protocol/triple`: keep HTTP/2, HTTP/3, unary, and streaming behavior
  isolated in tests where possible.

## Pull Request Notes

- Include the issue reference in the PR body.
- Mention local validation commands and any checks that were intentionally left
  to CI.
- Do not broaden a PR to unrelated cleanup just because nearby code looks old.
