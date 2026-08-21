# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html);
while on `v0.x`, breaking changes bump the minor version.

## [Unreleased]

### Fixed
- Accept `text/plain` content types case-insensitively when validating POST
  success responses.

### Added
- Runnable godoc examples covering client construction and common calls.
- Dependabot updates for GitHub Actions and Go modules.

### Changed
- CI runs with minimal permissions and cancels superseded runs.
- Test suite reaches full statement coverage.

## [0.1.4] - 2026-08-18

### Added
- Full decoding of the `/settings` response schema (`ServerSettings`).

## [0.1.3] - 2026-08-04

### Fixed
- POST endpoint responses are validated against the exact plain-text
  confirmation documented for each endpoint instead of accepting any body.
- `text/plain` responses from POST endpoints are accepted again after the
  JSON-only content type check introduced in 0.1.1.

## [0.1.2] - 2026-08-04

### Changed
- Default HTTP timeout raised from 10s to 30s.

## [0.1.1] - 2026-08-03

### Fixed
- Base URLs with malformed scheme prefixes are rejected, and credentials are
  redacted from the resulting error message.
- POST success responses with a non-JSON content type are rejected.

### Changed
- Decode errors no longer embed the response body.

## [0.1.0] - 2026-08-01

### Added
- Initial release: typed client for the official Palworld server REST API
  (`/v1/api`) covering server info, player list, settings, metrics,
  announcements, kick, ban, unban, save, shutdown and stop, with context
  support, configurable timeouts, response size caps, custom HTTP client
  injection and structured `APIError` values.
