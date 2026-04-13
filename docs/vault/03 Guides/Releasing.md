---
title: Releasing
type: guide
status: active
created: 2026-04-13
updated: 2026-04-13
tags:
  - guide
  - release
  - cicd
aliases:
  - Release Guide
  - Release Process
related:
  - "[[Contributing]]"
  - "[[02 Decisions/ADR-010 Goreleaser plus Cosign Keyless Signing]]"
  - "[[04 Operations/Security Posture]]"
  - "[[06 Project/Roadmap]]"
  - "[[Overview]]"
---

# Releasing

Releases are fully automated. No human runs a build command or edits a changelog. The pipeline flows from conventional commits on `main` through release-please and goreleaser to signed binaries and a Go module tag. This guide explains each stage so you know what to expect and how to verify a release.

## The release pipeline at a glance

```
conventional commits on main
        |
        v
release-please opens/updates a Release PR
  (version bump in go.mod, CHANGELOG.md update)
        |
        v
maintainer merges the Release PR
        |
        v
release-please pushes a vX.Y.Z tag
        |
        v
tag push triggers .github/workflows/release.yaml
        |
        v
goreleaser builds cross-platform binaries
  + cosign keyless signing + SLSA provenance
        |
        v
GitHub Release published with artefacts
        |
        v
go get github.com/tresic-cloud/intelligence-cloud-go@vX.Y.Z
  resolves within minutes
```

## Stage 1: Conventional commits drive the version

Every commit on `main` is parsed by [release-please](https://github.com/googleapis/release-please). The commit type determines the version bump:

| Commit type | Version bump |
|---|---|
| `feat:` | MINOR (`0.1.0` -> `0.2.0`) |
| `fix:` | PATCH (`0.2.0` -> `0.2.1`) |
| `BREAKING CHANGE:` or `!` | MAJOR (`0.2.1` -> `1.0.0`), or MINOR during `v0.x` |
| `docs:`, `chore:`, `test:`, etc. | No bump (but included in changelog) |

During the `v0.x` pre-release phase, breaking changes bump MINOR instead of MAJOR (`bump-minor-pre-major: true` in the release-please config). This aligns with [[02 Decisions/ADR-004 Pre-1.0 Experimental API Policy|ADR-004]] on pre-1.0 stability.

If you need your commit messages to be correct, see the [[Contributing|conventional commits guide]].

## Stage 2: The Release PR

When release-please detects commits that warrant a version bump, it opens (or updates) a Release PR. This PR contains:

- A version bump in `go.mod` (if applicable).
- An updated `CHANGELOG.md` with entries derived from the parsed commits.
- A bumped version string in `.release-please-manifest.json`.

The Release PR stays open and accumulates changes until a maintainer decides it is time to cut a release. You can continue merging feature PRs -- release-please will update the Release PR automatically.

**Review the Release PR carefully.** The changelog is auto-generated, but it is worth checking that the version bump is correct (especially around breaking changes) and that no sensitive information leaked into commit messages.

## Stage 3: Tagging

When the Release PR is merged, release-please pushes a `vX.Y.Z` tag to `main`. This tag is the trigger for the build pipeline.

## Stage 4: Goreleaser + cosign

The tag push triggers `.github/workflows/release.yaml`, which runs goreleaser. The workflow:

1. **Builds binaries** for six platform targets: `{linux, darwin, windows} x {amd64, arm64}`.
2. **Generates checksums** (`SHA256SUMS`).
3. **Signs with cosign keyless** using GitHub OIDC -- no long-lived signing keys exist. The signature is recorded in the Sigstore transparency log.
4. **Publishes a GitHub Release** with all artefacts attached: archives, checksums, signatures, and SLSA provenance metadata.

See [[02 Decisions/ADR-010 Goreleaser plus Cosign Keyless Signing]] for the design rationale and [[04 Operations/Security Posture]] for the broader supply-chain security model.

## Stage 5: Go module availability

Because the tag follows Go module conventions (`vX.Y.Z`), the Go module proxy picks it up automatically. Shortly after the release:

```sh
go get github.com/tresic-cloud/intelligence-cloud-go@vX.Y.Z
```

This resolves to the exact tagged commit. No additional publishing step is needed -- Go's module proxy discovers the tag via the Git repository.

## Verifying a release

### Verify binary signatures

Every release artefact is signed with cosign keyless. To verify:

```sh
# Download the binary and its signature
gh release download vX.Y.Z --repo tresic-cloud/intelligence-cloud-go \
  --pattern 'icctl_*_linux_amd64.tar.gz' \
  --pattern 'icctl_*_linux_amd64.tar.gz.sig' \
  --pattern 'icctl_*_linux_amd64.tar.gz.pem'

# Verify with cosign
cosign verify-blob \
  --certificate icctl_*_linux_amd64.tar.gz.pem \
  --signature icctl_*_linux_amd64.tar.gz.sig \
  --certificate-identity-regexp '^https://github.com/tresic-cloud/intelligence-cloud-go/' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  icctl_*_linux_amd64.tar.gz
```

The `--certificate-identity-regexp` ensures the signature came from the project's GitHub Actions workflow and not from an unrelated repository.

### Verify checksums

```sh
gh release download vX.Y.Z --repo tresic-cloud/intelligence-cloud-go --pattern 'checksums.txt'
sha256sum -c checksums.txt
```

### Verify the Go module

```sh
go mod verify
```

This checks that the downloaded module matches the checksum database (`sum.golang.org`).

## What to do when a release goes wrong

### The Release PR has the wrong version

If release-please chose the wrong version (e.g. it should be MAJOR but the breaking change footer was missing), do **not** edit the Release PR manually. Instead:

1. Close the Release PR without merging.
2. Push a corrective commit to `main` with the correct conventional-commit message (including the `BREAKING CHANGE:` footer).
3. Release-please will open a new Release PR with the correct version.

### A binary has a critical bug

If a released binary has a critical bug:

1. Fix the bug on `main` with a `fix:` commit.
2. Let release-please open a patch Release PR.
3. Merge the Release PR to cut a new patch release.

Do **not** delete or overwrite existing release tags. Consumers may have already pinned to the broken version, and deleting tags breaks the Go module proxy cache.

### The release workflow failed

Check the Actions tab for the failed workflow run. Common causes:

- goreleaser configuration error -- fix `.goreleaser.yaml` and push a corrective tag.
- cosign signing failure -- usually a transient GitHub OIDC issue. Re-run the workflow.
- Network timeout fetching dependencies -- re-run the workflow.

## Release checklist (for maintainers)

1. Verify all CI checks pass on `main`.
2. Review the Release PR: version bump is correct, changelog is accurate.
3. Merge the Release PR.
4. Wait for the release workflow to complete (typically 3--5 minutes).
5. Verify the GitHub Release page shows all six binary artefacts plus checksums and signatures.
6. Verify `go get ...@vX.Y.Z` resolves the new version.
7. Optionally verify a binary signature using the cosign command above.

## See also

- [[Contributing]] -- commit conventions that feed into this pipeline
- [[02 Decisions/ADR-010 Goreleaser plus Cosign Keyless Signing]] -- the ADR behind this release toolchain
- [[02 Decisions/ADR-004 Pre-1.0 Experimental API Policy]] -- pre-1.0 versioning policy
- [[04 Operations/Security Posture]] -- supply-chain security model
- [[06 Project/Roadmap]] -- planned release milestones
- [[Overview]] -- system architecture
