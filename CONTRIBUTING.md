# Contributing to palrest-go

Thanks for taking the time to contribute. This document covers the development
workflow for the library. Project conventions live in [AGENTS.md](AGENTS.md).

## Requirements

- Go 1.26 or newer (CI tests Go 1.26.x and 1.27.x).
- `make` and `git`.

## Getting Started

```bash
git clone https://github.com/TagamerStudio/palrest-go.git
cd palrest-go
make check
```

`make check` runs `go vet`, the pinned `golangci-lint` version and the full test
suite with the race detector. Run it before every commit.

## Development Commands

| Command      | Description                                                  |
|--------------|--------------------------------------------------------------|
| `make test`  | Tests with `-race` and verbose output.                       |
| `make lint`  | Runs the pinned `golangci-lint` via `go run`.                |
| `make vet`   | Runs `go vet ./...`.                                         |
| `make fmt`   | Installs `gofumpt`/`goimports` and formats the tree.         |
| `make fuzz`  | Runs the base URL fuzz target for 30 seconds.                |
| `make tidy`  | Runs `go mod tidy`.                                          |
| `make check` | `vet` + `lint` + `test`.                                     |

The project has no runtime dependencies; keep it that way unless a dependency is
clearly justified.

## Tests

- Use the standard `testing` package and table-driven tests where it fits.
- Put tests in `*_test.go` files next to the code they cover.
- New behavior needs tests; bug fixes should include a regression test.
- Fuzz seeds live in `fuzz_test.go`. Add seeds for new parsing edge cases.
- The REST contract fixtures live in `testdata/contract/<palworld-version>/`
  and are named after the Palworld game version. Record the REST API reference
  version and observed fields in the fixture README. Update
  `contractGameVersion` and `contractEndpoints` in `contract_test.go` when the
  schema changes, and keep old fixtures unless they are intentionally dropped.

## Code Conventions

- Write code, comments, tests and documentation in English.
- Every exported identifier needs a Go doc comment.
- Prefer `errors.New` for static errors and `%w` when wrapping.
- Add comments only when the code is not self-explanatory; document non-obvious
  protocol behavior and security decisions.
- Normalize or validate inputs at the boundary and keep the public API
  context-aware.

## Commits

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add a new endpoint
fix: reject malformed responses
refactor: simplify URL normalization
docs: clarify timeout behavior
test: cover redirect handling
chore: update tooling
```

Keep one logical change per commit and make sure `make check` passes first.
Breaking changes are allowed while on `v0.x`, but they must bump the minor
version and be documented.

## Pull Requests

- Describe the behavior change, not only the files touched.
- Link the related issue when one exists.
- Include tests and documentation updates in the same PR.
- Note user-visible changes so they can be added to `CHANGELOG.md` at release.
- Make sure CI is green before requesting review.

## Security

Do not report security vulnerabilities in public issues. See
[SECURITY.md](SECURITY.md) for the private reporting channel and scope.

## License

By contributing, you agree that your contributions are licensed under the
BSD 3-Clause license used by this project.
