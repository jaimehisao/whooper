# Security Policy

## Supported Versions

The latest `main` branch and latest tagged release are supported for security updates.

## Reporting a Vulnerability

If you discover a security issue, please report it privately:

- Open a private security advisory in GitHub, or
- Contact the maintainer directly through repository owner channels.

Please include:

- A clear description of the issue and impact.
- Reproduction steps or proof-of-concept.
- Any suggested remediation if available.

Do not publicly disclose vulnerabilities until a fix is available.

## Security Controls in CI

- Dependency vulnerability scanning (`govulncheck`).
- Secret scanning (`gitleaks`).
- Dependency update automation (Dependabot).
