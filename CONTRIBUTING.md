# Contributing

## Branch

Our default branch is `main`, which receives new features and bugfixes. Release
branches, e.g. `release-3.0`, will not received new features but bugfixes.
Opening a pull request for the release branches is what we called "backport".

## Title

The format of the title of PRs should be:

```
<type>: One line summary of the PR
```

The types are including(https://www.freecodecamp.org/news/how-to-write-better-git-commit-messages/):

- feat: a new feature is introduced with the changes
- fix: a bug fix has occurred
- chore: changes that do not relate to a fix or feature and don't modify src or test files (for example updating dependencies)
- refactor: refactored code that neither fixes a bug nor adds a feature
- docs: updates to documentation such as a the README or other markdown files
- style: changes that do not affect the meaning of the code, likely related to code formatting such as white-space, missing semi-- - colons, and so on.
- test: including new or correcting previous tests
- perf: performance improvements
- ci: continuous integration related
- build: changes that affect the build system or external dependencies
- revert: reverts a previous commit

If this PR is to backport to other branchs, e.g. `3.0`, then the title would be:

```
[3.0] <type>: One line summary of the PR
```

## Commit message

Each pull request **MUST** have a corresponding issue to explain what does this PR do. The format of message should be:

```
(A brief description about this PR.) Lorem ipsum dolor sit amet, consectetur
adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna
aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi
ut aliquip ex ea commodo consequat.

Fixes: #1000

Signed-off-by: Example <example@example.com>
```

The signed-off message will be appended automaticaly if the developers commit
with `-s` option, for example, `git commit -s`.

## AI coding agents

A repository-local onboarding skill lives at
`.agents/skills/dubbogo-onboarding/SKILL.md`. It uses the
[Agent Skills open format](https://agentskills.io/specification), and
`.agents/skills/` is the shared discovery path used by several agent clients
(Codex, Cursor, GitHub Copilot / VS Code, Gemini CLI, among others), so those
tools pick it up without any per-tool configuration. Contributors using other
tools can load the file explicitly.

The skill is a map of the repository and its validation commands, not an
authoritative specification: when it disagrees with the source tree, the
source tree wins. Please update it when Makefile targets, module layout, or
the extension registration APIs change.
