# Tagamer Palworld REST Client (Go)

[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev)
[![License: BSD 3-Clause](https://img.shields.io/badge/License-BSD%203--Clause-blue.svg)](LICENSE)

Typed Go client for the official [Palworld server REST API](https://docs.palworldgame.com/api/rest-api/palwold-rest-api)
(`/v1/api`): server info, player list, settings, world snapshot (`/game-data`,
with discriminated `CharacterActor`/`PalBoxActor` kinds), metrics, announce,
kick, ban, unban, save, shutdown and stop. All methods take a
`context.Context`.

Server must run with `RESTAPIEnabled=True`. The API targets recent server
versions: `/settings`, `/metrics`, `/unban`, `/stop` and `/game-data` appeared
in later documentation lines (all present since `1.0.2`), and `/game-data`
additionally requires launching the server with `-enable-gamedata-api`.

## Install

```bash
go get github.com/tagamer-net/palrest-go
```

## Usage

```go
import "github.com/tagamer-net/palrest-go"

client, err := palrest.NewClient("127.0.0.1:8212", "admin-password")
if err != nil {
    // handle error
}
defer client.Close()

ctx := context.Background()

info, err := client.GetServerInfo(ctx)
if err != nil {
    // handle error
}

err = client.MakeAnnouncement(ctx, "Server restart in 5 minutes")
```

### Options

| Option              | Description                                                             |
|---------------------|-------------------------------------------------------------------------|
| `WithTimeout(d)`    | HTTP timeout (default: 30s; values `<= 0` are ignored). Only applies to the internally created `http.Client`. |
| `WithHTTPClient(c)` | Inject a custom `http.Client`; its own `Timeout` and redirect policy are used as-is, `WithTimeout` is ignored and `Close()` becomes a no-op. A `nil` client is ignored and falls back to the default internal one. The internal client never follows redirects and ignores environment proxies (`HTTP_PROXY`/`HTTPS_PROXY`). |
| `WithMaxResponseBytes(n)` | Maximum accepted GET response body size in bytes (default: 10 MiB); larger responses fail with an error. Applies to every GET endpoint, including `/game-data`. POST responses are validated against the documented plain-text confirmation of each endpoint under a fixed 4 KiB cap. |

`baseURL` must be a scheme + host (optionally with port); paths, query strings,
fragments, userinfo and invalid hostnames are rejected since the client always
calls `/v1/api` endpoints directly. The scheme is case-insensitive and
normalized to lowercase (`HTTP://` is accepted and becomes `http://`). The
default port is `8212` when omitted,
and ports outside 1–65535 are rejected.

> **Large responses:** `/game-data` returns a snapshot of **every actor in the
> world**, so on large servers it can exceed the default 10 MiB cap. If you
> get a "response body exceeds" error, raise the cap for this client, e.g.
> `palrest.WithMaxResponseBytes(64 << 20)`. No other endpoint grows without
> bound (player counts are capped by the server settings and the remaining
> responses are fixed-schema). Snapshot generation also takes time on large
> worlds, so you may need to raise the default 30s timeout too, e.g.
> `palrest.WithTimeout(60 * time.Second)`.

Basic auth is always the fixed `admin` user; only the password is required.
The password is used exactly as provided (it is not trimmed), so pass it
without accidental leading or trailing whitespace.

Messages passed to `MakeAnnouncement`, `KickPlayer`, `BanPlayer` and
`ShutdownServer` are trimmed of leading/trailing whitespace; empty messages
are omitted from the payload (and rejected entirely for announcements).

`ShutdownServer` accepts a non-negative `waitTime`; the official documentation
does not define the behavior of `0` (observed server behavior is immediate shutdown). Use `StopServer` for a force stop.

## Security

The Palworld REST API is plain HTTP with basic auth: the admin password is
sent base64-encoded, which is **not** encrypted. The official documentation
recommends using the API on LAN only. For any access beyond a trusted LAN,
terminate TLS with a reverse proxy (e.g., Caddy or nginx) in front of the
server and pass the `https://` URL to `NewClient`. This library never
disables TLS certificate verification.

## Errors

HTTP errors are returned as `*APIError` with `StatusCode`, `Method`, `Path`
and `ResponseBody`:

```go
_, err := client.GetServerMetrics(ctx)
var apiErr *palrest.APIError
if errors.As(err, &apiErr) {
    // apiErr.StatusCode, apiErr.Method, apiErr.Path,
    // apiErr.ResponseBody (decoded JSON or raw text, for logging)
}
```

The official API documents `200`/`400`/`401` for most
endpoints (`/game-data` documents only `200`/`401`); error bodies are
undocumented and may vary between versions, and are capped at 1 KiB for
logging. GET responses with an empty or JSON-`null` body are treated as
protocol errors. POST success responses must be exactly the plain-text
confirmation documented for the endpoint (`/announce` -> "The message was
announced.", `/kick` -> "The player was kicked.", `/ban` -> "The player was
banned.", `/unban` -> "The player was unbanned.", `/save` -> "Successfully
saved the world.", `/shutdown` -> "The server will shutdown.", `/stop` ->
"The server force stopped."). An empty, JSON or otherwise divergent body, or
a non-`text/plain` content type, is treated as an error (the content-type
check runs before the body check so a 200 error page with an empty body is
still detected, never reported as success). The internally
created client never follows HTTP redirects (3xx responses surface as errors);
if you inject a client via `WithHTTPClient`, its redirect policy applies.

## Development

```bash
go test ./...
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.12.2 run --config .golangci.yml ./...
```

## License

BSD 3-Clause — see [LICENSE](LICENSE).
