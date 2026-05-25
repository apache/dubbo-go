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

# Installing dubbo-go Agent Skills for Codex

## Quick install

Tell Codex:

```text
Fetch and follow instructions from https://raw.githubusercontent.com/apache/dubbo-go/main/.agents/.codex/INSTALL.md
```

## Manual installation

Install the skills globally when you want Codex to use them outside an Apache dubbo-go checkout.

```bash
git clone https://github.com/apache/dubbo-go.git ~/.codex/dubbo-go
mkdir -p ~/.agents/skills
ln -s ~/.codex/dubbo-go/.agents/skills ~/.agents/skills/dubbo-go
```

Restart Codex.

### Windows

```powershell
git clone https://github.com/apache/dubbo-go.git "$env:USERPROFILE\.codex\dubbo-go"
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.agents\skills"
cmd /c mklink /J "$env:USERPROFILE\.agents\skills\dubbo-go" "$env:USERPROFILE\.codex\dubbo-go\.agents\skills"
```

## Updating

```bash
cd ~/.codex/dubbo-go
git pull --ff-only
```

Restart Codex after updating.

## Usage

The installed skills cover scaffolding, custom SPI extensions, Java interoperability, runtime debugging, conceptual guidance, and migration for dubbo-go v3 — plus a contributor-focused skill for modifying the `apache/dubbo-go` repository itself.
