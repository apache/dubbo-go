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

## Install

Install these skills globally when the user asks to install dubbo-go skills for Codex.

```bash
if [ -d ~/.codex/dubbo-go/.git ]; then
  git -C ~/.codex/dubbo-go pull --ff-only
else
  git clone https://github.com/apache/dubbo-go.git ~/.codex/dubbo-go
fi

mkdir -p ~/.agents/skills
rm -rf ~/.agents/skills/dubbo-go
ln -s ~/.codex/dubbo-go/.agents/skills ~/.agents/skills/dubbo-go
```

Restart Codex after installation.

### Windows

```powershell
$repo = "$env:USERPROFILE\.codex\dubbo-go"
if (Test-Path "$repo\.git") {
  git -C $repo pull --ff-only
} else {
  git clone https://github.com/apache/dubbo-go.git $repo
}

New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\.agents\skills"
Remove-Item -Recurse -Force "$env:USERPROFILE\.agents\skills\dubbo-go" -ErrorAction SilentlyContinue
cmd /c mklink /J "$env:USERPROFILE\.agents\skills\dubbo-go" "$env:USERPROFILE\.codex\dubbo-go\.agents\skills"
```

Restart Codex after installation.

## Update

```bash
cd ~/.codex/dubbo-go
git pull --ff-only
rm -rf ~/.agents/skills/dubbo-go
ln -s ~/.codex/dubbo-go/.agents/skills ~/.agents/skills/dubbo-go
```

Restart Codex after updating.

## Usage

The installed skills cover scaffolding, custom SPI extensions, Java interoperability, runtime debugging, conceptual guidance, and migration for dubbo-go v3 — plus a contributor-focused skill for modifying the `apache/dubbo-go` repository itself.
