# Security Policy

## Supported Versions

BlanketOps Environments CLI is pre-1.0 and moves fast — only the **latest
released version** receives security fixes. There's no long-term support
branch yet; upgrade to the latest release to pick up a fix.

| Version        | Supported          |
| -------------- | ------------------- |
| Latest release | :white_check_mark:  |
| Older releases | :x:                 |

This will change once the project reaches a 1.0 release with a defined
support window.

## Release Integrity

Every tagged release is signed and attested — verify before you trust a
binary, especially if you didn't build it yourself:

```bash
# Verify the Cosign signature
cosign verify-blob --certificate-identity-regexp ".*" \
  --cert-oidc-issuer "https://token.actions.githubusercontent.com" \
  --signature bin/bops-env-static.sig bin/bops-env-static

# Verify the GitHub build-provenance attestation
gh attest verify bin/bops-env-static --owner blanketops
```

## Reporting a Vulnerability

Please report security vulnerabilities privately — **do not open a public
GitHub issue**.

Use [GitHub's private vulnerability reporting](https://github.com/blanketops/environments-cli/security/advisories/new)
for this repository. If that's unavailable, contact a maintainer directly
through GitHub.

Include what you'd include in any good report: affected version/commit,
reproduction steps, and impact. We'll acknowledge new reports within a few
business days. This is a small, actively-developed project — there's no
formal SLA yet, but valid reports are prioritized over other work.

Once a fix is available, we'll coordinate disclosure timing with you and
credit you in the release notes unless you'd prefer otherwise.
