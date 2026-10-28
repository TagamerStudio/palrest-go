# Security Policy

## Supported Versions

Only the latest release line receives security fixes.

| Version | Supported          |
| ------- | ------------------ |
| 0.3.x   | :white_check_mark: |
| < 0.3   | :x:                |

## Reporting a Vulnerability

Please do not report security vulnerabilities in public issues.

Use GitHub's private vulnerability reporting form:

<https://github.com/TagamerStudio/palrest-go/security/advisories/new>

If the form is unavailable, open an issue titled "Security contact request"
without any vulnerability details and a maintainer will provide a private
channel.

Include, when possible:

- the affected version of this library;
- the Go version and operating system;
- the Palworld server version and REST API reference version;
- a minimal reproduction;
- the potential impact;
- relevant logs with credentials and personal data removed.

Never include the Palworld admin password, server addresses, player data or
other secrets in a report.

## Response

Reports are handled on a best-effort basis. We aim to acknowledge valid reports
within 3 business days and to coordinate disclosure after a fix is available.
Please allow time for a release before disclosing the issue publicly.

## Scope

This policy covers vulnerabilities in the palrest-go library, including URL
handling, credential management, TLS behavior, response size limits and
request/response validation.

The following are out of scope:

- vulnerabilities in the Palworld dedicated server itself; report those to
  Pocketpair;
- the inherent lack of transport encryption in the Palworld REST API. The API
  uses plain HTTP with Basic Auth, which only base64-encodes the admin password.
  Run it on a trusted LAN or behind a TLS-terminating reverse proxy and pass an
  `https://` base URL to `NewClient`.

## Secure Usage

- Keep the REST API on a trusted network or behind a TLS reverse proxy.
- The password is used exactly as provided; avoid accidental leading or trailing
  whitespace.
- Treat `APIError.ResponseBody` as diagnostic data and avoid logging it if it
  may contain sensitive information.
- When injecting an HTTP client with `WithHTTPClient`, configure its timeout,
  redirect, proxy and TLS policies safely.
