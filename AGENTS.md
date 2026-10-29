# AGENTS.md — palrest-go

Typed Go client for the official Palworld server REST API.

## Conventions

- **Language:** Go 1.26+
- **Module:** `github.com/TagamerStudio/palrest-go` (package `palrest`)
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
- Run `make check` before committing (vet + lint + test).

## Commit guidelines

Conventional Commits (`feat:`, `fix:`, `refactor:`, `build:`, `ci:`, `chore:`,
`docs:`, `test:`), one logical change per commit, `make check` before
committing. Direct pushes to `main` are blocked; open a pull request and wait
for `CI success`.

## Versioning

Published with semver tags (currently `v0.3.3`). Breaking changes bump the
minor version while on `v0.x`. Compatibility with Palworld game versions is
documented in `COMPATIBILITY.md`; the library version is independent of the
game version.
