# Compatibility

`palrest-go` targets Palworld game versions. The REST API reference version is
recorded as the schema source and may lag behind the game.

| palrest-go | Palworld | REST API reference | Fixtures | Status |
|------------|----------|--------------------|----------|--------|
| v0.3.x     | v1.0.5   | 1.0.4              | testdata/contract/v1.0.5 | documented |

Compatibility target: Palworld v1.0.5.

## Policy

- The Palworld game version is the primary compatibility axis.
- Fixtures are named after the Palworld game version they target.
- Hotfixes without API changes do not require a new fixture; update the
  compatibility target and status in this file instead.
- `documented` means based on the reference and patch notes; `tested` means
  verified against a live server.
- Fields observed on game servers but missing from the reference are part of
  the compatibility surface and stay in the Go types and fixtures.
- The REST API reference version documents schema provenance, not a
  compatibility promise.

## Notes

- v1.0.5 is a hotfix release with no documented REST API changes.
- The current fixtures were built from the official API reference 1.0.4 plus
  observed fields; they are not a live capture of a v1.0.5 server.
- Fixture provenance is recorded in `testdata/contract/v1.0.5/README.md`.
