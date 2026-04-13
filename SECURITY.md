# Security Policy

## Supported versions

| Version | Supported |
|---------|-----------|
| Latest v0.x | Yes |
| Older releases | No |

Only the latest release in the current major version line receives security
patches. Upgrade to the latest release before reporting a vulnerability.

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.**

### Preferred: GitHub Security Advisories

Use [GitHub Security Advisories](https://github.com/tresic-cloud/intelligence-cloud-go/security/advisories/new)
to report vulnerabilities privately. This is the preferred channel because it
keeps the report confidential until a fix is available.

### Alternative: email

If you cannot use GitHub Security Advisories, send an email to
**security@tresic.cloud** with the subject line
`[intelligence-cloud-go] Vulnerability Report`.

## Response timeline

| Stage | SLA |
|-------|-----|
| Acknowledgement | Within 48 hours |
| Triage and severity assessment | Within 5 business days |
| Fix development | Depends on severity |
| Patch release | As soon as the fix is verified |

We will keep you informed of progress at each stage.

## Fix and disclosure process

1. The vulnerability is triaged and a severity is assigned (CVSS where applicable).
2. A fix is developed on a private branch.
3. A patch release is published with CVE references in the release notes and
   CHANGELOG.
4. A public advisory is published on the repository after the fix is released.

## Supply-chain verification

Release binaries are signed with **cosign keyless** signing using GitHub OIDC.
No long-lived signing keys exist. Verify any release binary with:

```bash
cosign verify-blob \
  --certificate-identity-regexp "github.com/tresic-cloud/intelligence-cloud-go" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --bundle <artifact>.sig.bundle \
  <artifact>
```

Go module integrity is additionally verified by the Go checksum database
(`sum.golang.org`).

## Further reading

See [docs/vault/04 Operations/Security Posture.md](./docs/vault/04%20Operations/Security%20Posture.md)
for the full security model, including secrets lifecycle, redaction rules,
transport security, and supply-chain details.
