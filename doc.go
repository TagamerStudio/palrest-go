// Package palrest provides a typed HTTP client for the official Palworld
// dedicated server REST API (/v1/api).
//
// # Overview
//
// The client covers every documented endpoint: server info, player list,
// settings, world snapshot (/game-data), metrics, announcements, kick, ban,
// unban, save, shutdown and stop. Endpoint methods accept a context.Context.
// GET methods return values decoded from the documented JSON schemas, while
// POST methods return an error after validating the server confirmation.
//
// # Usage
//
// Create a client once and reuse it across requests:
//
//	client, err := palrest.NewClient("127.0.0.1:8212", "admin-password")
//	if err != nil {
//		return err
//	}
//	defer client.Close()
//
//	info, err := client.GetServerInfo(ctx)
//
// Behavior is adjusted with the options accepted by NewClient:
// WithTimeout, WithMaxResponseBytes and WithHTTPClient.
//
// # Errors
//
// HTTP failures are returned as *APIError carrying the status code, method,
// path and response body for logging. GET responses with an empty or JSON-null
// body are treated as protocol errors, and POST endpoints validate the exact
// plain-text confirmation documented for each one.
//
// # Security
//
// The REST API is plain HTTP with basic auth: the admin password is sent
// base64-encoded, which is not encryption. Keep usage on a trusted LAN, or
// terminate TLS with a reverse proxy and pass an https:// base URL to
// NewClient. The internally created client uses standard TLS verification by
// default, never follows redirects and ignores environment proxies.
// WithHTTPClient delegates timeout, redirect, proxy and TLS policy to the
// injected client, so callers must configure those settings safely.
package palrest
