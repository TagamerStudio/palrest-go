# Palworld v1.0.5 contract fixtures

- Palworld game version: v1.0.5
- REST API reference source: 1.0.4
- v1.0.5 is a hotfix release with no documented REST API changes.
- `settings.json` covers every field in `ServerSettings`, including observed
  fields that are absent from the reference, such as `BaseCampMaxNumInGuild`.
- `game-data.json` contains complete `Character` and `PalBox` actor payloads.
- Payload values are illustrative, not a live capture.

When a game release changes the API, add a new versioned directory and update
`COMPATIBILITY.md`.
