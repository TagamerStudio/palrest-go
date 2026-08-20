# AGENTS.md — palrest-go

Typed Go client for the official Palworld server REST API.

## Conventions

- **Language:** Go 1.25+
- **Module:** `github.com/tagamer-net/palrest-go` (package `palrest`)
- **Library only:** no CLI, no config loading, no env vars. Constructors take
  everything explicitly (URL, credentials, options). Environment reads are the
  caller's responsibility.
- Error bodies of the Palworld REST API are undocumented — do not rely on or
  expose them as structured data. `APIError` keeps the decoded body only for
  logging.

## Coding conventions

- Development entirely in English; no code comments unless necessary.
- Prefer `errors.New` over `fmt.Errorf` for static strings; use `%w` when wrapping.
- All public functions/types need Go doc comments.
- Tests: standard `testing` package, table-driven, `*_test.go` alongside source.
- Run `make check` before committing (lint + test).

## Commit guidelines

Conventional Commits (`feat:`, `fix:`, `refactor:`, `chore:`, `docs:`, `test:`),
one logical change per commit, `make check` before committing.

## Versioning

Published with semver tags (currently `v0.1.4`). Breaking changes bump the
minor version while on `v0.x`.
