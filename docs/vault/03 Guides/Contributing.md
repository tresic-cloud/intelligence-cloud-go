---
title: Contributing
type: guide
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - guide
  - contributing
  - conventions
aliases:
  - Contributing Guide
  - Dev Setup
related:
  - "[[Releasing]]"
  - "[[Overview]]"
  - "[[06 Project/Constitution Compliance]]"
  - "[[02 Decisions/ADR-002 Cobra and Viper for CLI]]"
  - "[[02 Decisions/ADR-010 Goreleaser plus Cosign Keyless Signing]]"
---

# Contributing

This guide covers everything you need to set up a development environment, make changes that meet the project's quality bar, and land a pull request. The project follows the Tresic constitution strictly -- see [[06 Project/Constitution Compliance]] for how each principle is met.

## Dev environment setup

### Prerequisites

| Tool | Version | Purpose |
|---|---|---|
| Go | 1.22+ (1.23 recommended) | Build and test |
| golangci-lint | latest | Lint |
| govulncheck | latest | Vulnerability scanning |
| git | 2.x+ | Version control |

### Clone and verify

```sh
git clone https://github.com/tresic-cloud/intelligence-cloud-go.git
cd intelligence-cloud-go
go mod download
```

### Run the test suite

The project does not yet have a top-level `Makefile` (deferred to a later stream), so run the tools directly:

```sh
# Unit tests with race detection and coverage
go test -race -cover ./...

# Lint
golangci-lint run ./...

# Vulnerability scan
govulncheck ./...
```

All three must pass before you open a pull request. CI runs the same commands -- there should be no surprises.

## Conventional commits

**This is load-bearing.** The project uses [release-please](https://github.com/googleapis/release-please) to automate version bumps and changelog generation. Release-please parses commit messages on `main` to decide whether to cut a MAJOR, MINOR, or PATCH release. If your commit message does not follow the convention, it will not appear in the changelog and may cause a version bump to be skipped.

See [[02 Decisions/ADR-010 Goreleaser plus Cosign Keyless Signing|ADR-010]] and [[Releasing]] for the full release pipeline.

### Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types and their version impact

| Type | Description | Version bump |
|---|---|---|
| `feat` | A new feature visible to consumers | **MINOR** |
| `fix` | A bug fix | **PATCH** |
| `docs` | Documentation-only changes | none |
| `chore` | Maintenance (deps, config) | none |
| `test` | Adding or fixing tests | none |
| `refactor` | Code change that neither fixes a bug nor adds a feature | none |
| `perf` | Performance improvement | none |
| `build` | Build system or external dependency changes | none |
| `ci` | CI configuration changes | none |
| `style` | Code style (formatting, semicolons, etc.) | none |

### Breaking changes

A breaking change triggers a **MAJOR** version bump (or MINOR during `v0.x` pre-releases). Signal it in one of two ways:

```
feat!: remove deprecated ListAll method

BREAKING CHANGE: ListAll has been removed. Use the Iterator[T]
returned by List() instead.
```

Either the `!` after the type or a `BREAKING CHANGE:` footer (or both) will trigger the major bump. Use the footer for a longer explanation.

### Examples

```
feat(sdk): add WithMaxItems option for paginated lists

fix(cli): correct exit code for auth failures

docs: update quickstart with new base URL

chore(deps): bump oapi-codegen to v2.4.0

refactor(transport): extract retry logic into standalone RoundTripper

BREAKING CHANGE: RetryPolicy.MaxDelay field renamed to MaxBackoff.

perf(pagination): reduce allocations in Iterator.Next

test(errors): add fuzz tests for error classification

ci: add govulncheck to CI pipeline
```

### PR title convention

PR titles are also linted by `amannn/action-semantic-pull-request` in CI. Since the project squash-merges PRs, the PR title becomes the commit message on `main`. Write your PR title in conventional-commit format.

## Branch naming

All branches must follow the naming convention from the project constitution:

| Branch type | Pattern | Example |
|---|---|---|
| Feature | `feature/{JIRA-KEY}-{short-description}` | `feature/IDB-31-get-reseller-user` |
| Bugfix | `bugfix/{JIRA-KEY}-{short-description}` | `bugfix/IDB-45-fix-pagination` |
| Hotfix | `hotfix/{JIRA-KEY}-{short-description}` | `hotfix/IDB-99-critical-auth-fix` |
| Release | `release/v{version}` | `release/v1.2.0` |

Rules:

- All feature, bugfix, and hotfix branches **must** include the Jira ticket key.
- Use lowercase with hyphens (kebab-case).
- Keep descriptions short but meaningful (3--5 words max).

## TDD discipline

The project mandates Test-Driven Development. This is not aspirational -- it is a hard requirement enforced in code review.

### The cycle

1. **Write a failing test first** -- before any implementation code exists.
2. **Implement the minimum code** to make the test pass.
3. **Refactor** while keeping the test green.
4. **Repeat** until the feature is complete.

### Coverage target

The project requires **at least 80% line coverage** on every package. Check locally:

```sh
go test -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

CI will reject PRs that drop coverage below the threshold. The [[05 Reference/Error Hierarchy|error hierarchy]], [[05 Reference/Pagination|pagination iterator]], and [[01 Architecture/HTTP Transport|HTTP transport]] are particularly sensitive areas where coverage matters.

### What counts as a violation

- Writing implementation before tests.
- Tests that assert nothing meaningful (e.g. `if err != nil { t.Fatal(err) }` with no positive assertions).
- Skipping tests for "simple" code.
- Coverage below 80%.

## Code style and linting

The project uses `golangci-lint` with a project-level configuration. Key linters enabled:

- `gofmt` / `goimports` -- formatting
- `govet` -- correctness
- `errcheck` -- unchecked errors
- `staticcheck` -- static analysis
- `gosec` -- security patterns
- `revive` -- style

Run lint locally before pushing:

```sh
golangci-lint run ./...
```

## Pull request process

### Before you open a PR

1. All tests pass locally (`go test -race -cover ./...`).
2. Lint is clean (`golangci-lint run ./...`).
3. No known vulnerabilities (`govulncheck ./...`).
4. Coverage meets the 80% threshold.
5. Your branch name follows the convention above.
6. Your PR title is in conventional-commit format.

### Required checks

CI runs the following on every PR:

- `go test -race -cover ./...` (unit tests)
- `golangci-lint run ./...` (lint)
- `govulncheck ./...` (vulnerability scan)
- Cross-platform build verification (Linux, macOS, Windows x amd64/arm64)
- PR title conventional-commit lint

All checks must pass before merge.

### Review requirements

- At least **one approving review** from a CODEOWNERS team member.
- Reviewers will check for TDD discipline (tests exist and came first), error handling completeness, and documentation.
- PRs are squash-merged. The PR title becomes the commit message on `main`, which is why the conventional-commit PR title format matters -- it feeds directly into release-please.

### After merge

Once your PR is squash-merged to `main`, release-please picks up the conventional commit and (if it is a `feat` or `fix`) opens or updates a Release PR with the version bump and changelog entry. You do not need to do anything further. See [[Releasing]] for the full flow.

## Project structure

Familiarise yourself with the module layout before diving in. The [[Overview|architecture overview]] has the full tree, but the key areas are:

| Path | What lives there |
|---|---|
| `intelligencecloud/` (root package) | Client, options, errors, retry, pagination, telemetry |
| `auth/` | CredentialProvider interface + adapters |
| `internal/generated/` | oapi-codegen output -- do not edit by hand |
| `internal/transport/` | RoundTripper composing auth, retry, OTel, redaction |
| `cmd/icctl/` | CLI binary (Cobra + Viper) |

## See also

- [[Releasing]] -- what happens after your code reaches `main`
- [[06 Project/Constitution Compliance]] -- how each constitution principle is met
- [[Overview]] -- system architecture
- [[02 Decisions/ADR-002 Cobra and Viper for CLI]] -- why Cobra + Viper
- [[05 Reference/Error Hierarchy]] -- the typed error hierarchy your code should use
