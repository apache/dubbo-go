Contributing to Dubbogo

## 1. Branch

  >- The name of branches `SHOULD` be in the format of `feature/xxx`.
  >- You `SHOULD` checkout a new branch after a feature branch already being merged into upstream, `DO NOT` commit in the old branch.

## 2. Pull Request

### 2.1. Title Format

The pr head format is `<head> <subject>`. The title should be simpler to show your intent.

The title format of the pull request `MUST` follow the following rules:

  >- Start with `Doc:` for adding/formatting/improving docs.
  >- Start with `Mod:` for formatting codes or adding comment.
  >- Start with `Fix:` for fixing bug, and its ending should be ` #issue-id` if being relevant to some issue.
  >- Start with `Imp:` for improving performance.
  >- Start with `Ftr:` for adding a new feature.
  >- Start with `Add:` for adding struct function/member.
  >- Start with `Rft:` for refactoring codes.
  >- Start with `Tst:` for adding tests.
  >- Start with `Dep:` for adding depending libs.
  >- Start with `Rem:` for removing feature/struct/function/member/files.

## 3. Code Style

### 3.1 log

> 1 when logging the function's input parameter, you should add '@' before input parameter name.
