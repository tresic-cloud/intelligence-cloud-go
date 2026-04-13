# Contributing

Contributions are welcome. This document covers the essentials for getting
started. For deeper detail, see
[docs/vault/03 Guides/Contributing.md](./docs/vault/03%20Guides/Contributing.md).

## Dev setup

```sh
git clone https://github.com/tresic-cloud/intelligence-cloud-go.git
cd intelligence-cloud-go
go mod download
make test
make lint
```

Prerequisites: Go 1.25+, golangci-lint, govulncheck, staticcheck.

## Useful Makefile targets

| Target | Purpose |
|--------|---------|
| `make test` | Run tests with race detector and coverage |
| `make lint` | Run golangci-lint |
| `make vuln` | Run govulncheck |
| `make static` | Run go vet + staticcheck |
| `make codegen` | Regenerate code from OpenAPI spec |
| `make bench` | Run benchmarks |
| `make docs` | Generate CLI reference |
| `make coverage` | Open HTML coverage report |
| `make clean` | Remove build artifacts |

## Conventional commits

**This is load-bearing.** The project uses
[release-please](https://github.com/googleapis/release-please) to automate
version bumps and changelog generation. Commit messages on `main` drive the
version number and changelog entries.

### Format

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

### Types and version impact

| Type | Description | Version bump |
|------|-------------|--------------|
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

A breaking change triggers a **MAJOR** version bump (or MINOR during `v0.x`).
Signal it with either:

- An `!` after the type: `feat!: remove deprecated method`
- A `BREAKING CHANGE:` footer in the commit body

Both can be used together. Use the footer for a longer explanation.

## Branch naming

All branches must follow the naming convention from the project constitution:

| Branch type | Pattern | Example |
|-------------|---------|---------|
| Feature | `feature/{JIRA-KEY}-{short-description}` | `feature/IDB-31-get-reseller-user` |
| Bugfix | `bugfix/{JIRA-KEY}-{short-description}` | `bugfix/IDB-45-fix-pagination` |
| Hotfix | `hotfix/{JIRA-KEY}-{short-description}` | `hotfix/IDB-99-critical-auth-fix` |
| Release | `release/v{version}` | `release/v1.2.0` |

Rules:
- All feature, bugfix, and hotfix branches **must** include the Jira ticket key.
- Use lowercase with hyphens (kebab-case).
- Keep descriptions short but meaningful (3-5 words max).

## TDD discipline

The project mandates Test-Driven Development. This is a hard requirement
enforced in code review.

1. **Write a failing test first** -- before any implementation code exists.
2. **Implement the minimum code** to make the test pass.
3. **Refactor** while keeping the test green.
4. **Repeat** until the feature is complete.

Coverage target: **>= 80% line coverage** on every package.

```sh
make test
make coverage   # opens HTML report
```

## Pull request process

1. All tests pass locally (`make test`).
2. Lint is clean (`make lint`).
3. No known vulnerabilities (`make vuln`).
4. Coverage meets the 80% threshold.
5. Branch name follows the convention above.
6. PR title is in conventional-commit format (CI enforces this).

PRs are squash-merged. The PR title becomes the commit message on `main`, which
feeds directly into release-please. At least one approving review from a
CODEOWNERS team member is required.

## Further reading

- [Contributing (extended)](./docs/vault/03%20Guides/Contributing.md) -- full
  guide with project structure, code style, and review expectations
- [Architecture overview](./docs/vault/01%20Architecture/Overview.md)
- [Security policy](./SECURITY.md)
