// Package palrest provides a typed HTTP client for the PalServer REST API.
//
// Supports server info, player listing, settings, world snapshot, metrics, and
// administrative actions (announce, kick, ban, shutdown, etc.) via basic-auth
// authentication. All methods accept a context.Context.
package palrest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Client configuration
// ---------------------------------------------------------------------------

const (
	defaultPort     = "8212"
	defaultTimeout  = 10 * time.Second
	defaultUsername = "admin"
	apiPrefix       = "/v1/api"

	maxResponseBytes  = 10 << 20
	maxErrorBodyBytes = 1 << 10
	drainLimitBytes   = 64 << 10

	// postResponseLimitBytes caps success response bodies of POST endpoints;
	// they are validated as empty or JSON and never need the general cap.
	postResponseLimitBytes = 4 << 10

	userAgent = "palrest-go"
)

// Option is a configuration function applied to the Client.
type Option func(*Client)

// WithTimeout sets the HTTP timeout used by the internally created http.Client.
// It has no effect when a custom client is injected via WithHTTPClient; the
// injected client's own Timeout is used as-is. Values <= 0 are ignored.
func WithTimeout(timeout time.Duration) Option {
	return func(c *Client) {
		if timeout > 0 {
			c.timeout = timeout
		}
	}
}

// WithMaxResponseBytes sets the maximum accepted response body size in bytes.
// Larger responses fail with an error. Values <= 0 are ignored. The default
// is 10 MiB.
func WithMaxResponseBytes(maxBytes int) Option {
	return func(c *Client) {
		if maxBytes > 0 {
			c.maxBodyBytes = maxBytes
		}
	}
}

// WithHTTPClient injects a custom http.Client (used in tests). When set,
// Close becomes a no-op and WithTimeout is ignored. The injected client's own
// redirect policy is used as-is; the internally created client never follows
// redirects and never uses environment proxies.
// A nil client is ignored and falls back to the default internal client
// (WithTimeout applies and Close is effective), matching the behavior of not
// passing the option.
// Use this to configure TLS (e.g., self-signed certs common in LAN deployments).
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.client = httpClient
	}
}

// ---------------------------------------------------------------------------
// RESTClient interface & implementation
// ---------------------------------------------------------------------------

// RESTClient defines the contract for interacting with the PalServer REST API.
type RESTClient interface {
	Close() error
	GetServerInfo(ctx context.Context) (*ServerInfo, error)
	GetPlayerList(ctx context.Context) (*PlayerList, error)
	GetServerSettings(ctx context.Context) (*ServerSettings, error)
	GetServerMetrics(ctx context.Context) (*ServerMetrics, error)
	GetGameData(ctx context.Context) (*GameData, error)
	MakeAnnouncement(ctx context.Context, message string) error
	KickPlayer(ctx context.Context, userID, message string) error
	BanPlayer(ctx context.Context, userID, message string) error
	UnbanPlayer(ctx context.Context, userID string) error
	SaveServerState(ctx context.Context) error
	ShutdownServer(ctx context.Context, waitTime int, message string) error
	StopServer(ctx context.Context) error
}

// Compile-time assertion that Client implements RESTClient.
var _ RESTClient = (*Client)(nil)

// Client is the typed HTTP client for the PalServer REST API. Basic auth is
// always the fixed "admin" user; the constructor takes the REST API password.
type Client struct {
	baseURL      string
	password     string
	client       *http.Client
	timeout      time.Duration
	maxBodyBytes int
	ownClient    bool
}

// APIError represents HTTP errors returned by the REST API.
type APIError struct {
	StatusCode   int
	Method       string
	Path         string
	ResponseBody any
}

func (e *APIError) Error() string {
	return fmt.Sprintf("rest api error: status=%d method=%s path=%s body=%v", e.StatusCode, e.Method, e.Path, e.ResponseBody)
}

// NewClient creates a REST client with a normalized base URL and optional
// options. The password is used exactly as provided (it is not trimmed); pass
// it without accidental leading or trailing whitespace.
func NewClient(baseURL, password string, opts ...Option) (*Client, error) {
	if strings.TrimSpace(baseURL) == "" {
		return nil, errors.New("base URL is required")
	}
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("password is required")
	}

	normalized, err := normalizeBaseURL(baseURL, defaultPort)
	if err != nil {
		return nil, err
	}

	c := &Client{
		baseURL:      normalized,
		password:     password,
		timeout:      defaultTimeout,
		maxBodyBytes: maxResponseBytes,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.client == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.Proxy = nil
		c.client = &http.Client{
			Transport: transport,
			Timeout:   c.timeout,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}
		c.ownClient = true
	}

	return c, nil
}

// Close releases idle connections held by the internally created http.Client.
// It is a no-op when a custom client was injected via WithHTTPClient.
func (c *Client) Close() error {
	if c.ownClient {
		c.client.CloseIdleConnections()
	}
	return nil
}

func (c *Client) buildURL(endpoint string) string {
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}
	return c.baseURL + apiPrefix + endpoint
}

// normalizeBaseURL ensures a valid base URL with an http/https scheme and the
// default port when missing. Supports IPv6 hosts (with brackets); zone-scoped
// IPv6 addresses keep their percent-escaped zone delimiter in the result.
// Paths, query strings, fragments and userinfo are rejected because the client
// always targets the /v1/api endpoints directly. Hosts must be valid IP
// addresses or DNS hostnames.
func normalizeBaseURL(raw, defaultPort string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("base URL is required")
	}

	raw, err := withScheme(raw)
	if err != nil {
		return "", err
	}

	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid base URL %q: %w", redactUserinfo(raw), err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported scheme %q in base URL", u.Scheme)
	}
	if u.User != nil {
		return "", fmt.Errorf("base URL must not contain userinfo: %q", redactUserinfo(raw))
	}

	if u.Path != "" && u.Path != "/" {
		return "", fmt.Errorf("base URL must not contain a path: %q", raw)
	}
	if u.RawQuery != "" {
		return "", fmt.Errorf("base URL must not contain a query string: %q", raw)
	}
	if u.Fragment != "" {
		return "", fmt.Errorf("base URL must not contain a fragment: %q", raw)
	}

	host := u.Hostname()
	if host == "" {
		return "", fmt.Errorf("base URL missing host: %q", raw)
	}
	if !validHostname(host) {
		return "", fmt.Errorf("invalid hostname %q in base URL", host)
	}

	port := u.Port()
	if port == "" {
		port = defaultPort
	} else if port, err = validatePort(port); err != nil {
		return "", fmt.Errorf("%w in base URL", err)
	}

	// url.Parse unescapes %25 -> % in zone-scoped IPv6 hosts (e.g., %25eth0 becomes %eth0).
	// The rebuilt URL must re-encode % as %25 or the next parse of a request URL fails
	// with an invalid escape error.
	host = strings.ReplaceAll(host, "%", "%25")
	return fmt.Sprintf("%s://%s", u.Scheme, net.JoinHostPort(host, port)), nil
}

// withScheme prepends the default scheme to a base URL that omits it and
// rejects prefixes that look like a scheme missing the "//" delimiter (e.g.,
// "http:/host"), which url.Parse would otherwise rewrite as a bogus host.
func withScheme(raw string) (string, error) {
	if strings.Contains(raw, "://") {
		return raw, nil
	}
	if strings.HasPrefix(raw, "http:/") || strings.HasPrefix(raw, "https:/") {
		return "", errors.New("invalid base URL: malformed scheme (missing '//')")
	}
	return "http://" + raw, nil
}

// validHostname reports whether host is a valid IP address or DNS hostname.
// A single trailing dot (FQDN form) is accepted.
func validHostname(host string) bool {
	if _, err := netip.ParseAddr(host); err == nil {
		return true
	}
	if len(host) > 0 && host[len(host)-1] == '.' {
		host = host[:len(host)-1]
	}
	if len(host) == 0 || len(host) > 253 {
		return false
	}
	if !validHostnameLabels(host) {
		return false
	}
	return host[0] != '-' && host[len(host)-1] != '-'
}

// validHostnameLabels checks DNS label structure: non-empty labels of at most
// 63 characters containing only ASCII letters, digits or '-', with '-' only
// allowed inside a label.
func validHostnameLabels(host string) bool {
	labelLen := 0
	for i := 0; i < len(host); i++ {
		r := host[i]
		if r == '.' {
			if labelLen == 0 || host[i-1] == '-' {
				return false
			}
			labelLen = 0
			continue
		}
		if !validLabelChar(r, labelLen) {
			return false
		}
		labelLen++
		if labelLen > 63 {
			return false
		}
	}
	return true
}

// validLabelChar reports whether r may appear at position labelLen of a DNS
// label. A '-' is only allowed when the label is not empty.
func validLabelChar(r byte, labelLen int) bool {
	if r == '-' && labelLen == 0 {
		return false
	}
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-'
}

// redactUserinfo masks credentials in raw so error messages referencing the
// URL do not leak the password into logs.
func redactUserinfo(raw string) string {
	schemeEnd := strings.Index(raw, "://")
	if schemeEnd < 0 {
		return raw
	}
	at := strings.LastIndexByte(raw[schemeEnd+3:], '@')
	if at < 0 {
		return raw
	}
	return raw[:schemeEnd+3] + "xxxxx" + raw[schemeEnd+3+at:]
}

// validatePort checks that port is a number within the valid TCP range and
// returns it in canonical form without leading zeros. Only plain digit
// strings are accepted (RFC 3986 ports contain no sign).
func validatePort(port string) (string, error) {
	if port == "" || port[0] < '0' || port[0] > '9' {
		return "", fmt.Errorf("invalid port %q", port)
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		return "", fmt.Errorf("invalid port %q: %w", port, err)
	}
	if n < 1 || n > 65535 {
		return "", fmt.Errorf("port %q out of range (1-65535)", port)
	}
	return strconv.Itoa(n), nil
}

// ---------------------------------------------------------------------------
// Generic helpers
// ---------------------------------------------------------------------------

// request performs an HTTP call and returns the raw response body together
// with the response Content-Type. maxBytes bounds the accepted response body
// size for success responses; error bodies are read under maxErrorBodyBytes
// regardless. Error bodies and oversized success bodies are drained up to
// drainLimitBytes before the connection is released so it can be reused.
func (c *Client) request(ctx context.Context, method, path string, body any, maxBytes int) ([]byte, string, error) {
	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, "", fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	target := c.buildURL(path)
	req, err := http.NewRequestWithContext(ctx, method, target, reqBody)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.SetBasicAuth(defaultUsername, c.password)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("%s %s: %w", method, target, err)
	}

	if resp.StatusCode >= http.StatusMultipleChoices {
		bodyBytes, readErr := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodyBytes))
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, drainLimitBytes))
		_ = resp.Body.Close()
		if readErr != nil {
			return nil, "", fmt.Errorf("failed to read error response body: %w", readErr)
		}
		return nil, "", &APIError{
			StatusCode:   resp.StatusCode,
			Method:       method,
			Path:         path,
			ResponseBody: decodeResponseBody(bodyBytes),
		}
	}

	defer func() { _ = resp.Body.Close() }()

	if maxBytes == math.MaxInt {
		maxBytes = math.MaxInt - 1
	}
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxBytes)+1))
	if err != nil {
		return nil, "", fmt.Errorf("failed to read response body: %w", err)
	}

	if len(bodyBytes) > maxBytes {
		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, drainLimitBytes))
		return nil, "", fmt.Errorf("response body exceeds %d bytes", maxBytes)
	}

	return bodyBytes, resp.Header.Get("Content-Type"), nil
}

// decodeResponseBody best-effort decodes an error body for APIError logging.
// Error bodies are undocumented, so raw text is kept when JSON decoding fails.
// The caller caps the body at maxErrorBodyBytes before this runs.
func decodeResponseBody(body []byte) any {
	if len(body) == 0 {
		return nil
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return strings.TrimSpace(string(body))
	}
	return payload
}

func (c *Client) requestInto(ctx context.Context, method, path string, body, out any, maxBytes int) error {
	bodyBytes, contentType, err := c.request(ctx, method, path, body, maxBytes)
	if err != nil {
		return err
	}
	if len(bodyBytes) == 0 {
		return fmt.Errorf("%s %s: empty response body", method, path)
	}
	if bytes.Equal(bytes.TrimSpace(bodyBytes), []byte("null")) {
		return fmt.Errorf("%s %s: unexpected null response body", method, path)
	}
	if err := json.Unmarshal(bodyBytes, out); err != nil {
		if contentType == "" {
			contentType = "unknown"
		}
		return fmt.Errorf("failed to decode %s %s (content-type %q): %w", method, path, contentType, err)
	}
	return nil
}

func (c *Client) getInto(ctx context.Context, path string, out any, maxBytes int) error {
	return c.requestInto(ctx, http.MethodGet, path, nil, out, maxBytes)
}

// post performs an HTTP POST and validates the success response. Success
// bodies must be empty or JSON; a non-JSON body or a mismatched content type
// indicates a misbehaving proxy or an error page served with status 200 and
// is treated as a failure. Bodies are capped at postResponseLimitBytes.
func (c *Client) post(ctx context.Context, path string, body any) error {
	bodyBytes, contentType, err := c.request(ctx, http.MethodPost, path, body, postResponseLimitBytes)
	if err != nil {
		return err
	}
	trimmed := bytes.TrimSpace(bodyBytes)
	if contentType != "" && !strings.HasPrefix(contentType, "application/json") {
		return fmt.Errorf("POST %s: unexpected content-type %q", path, contentType)
	}
	if len(trimmed) == 0 {
		return nil
	}
	if !json.Valid(trimmed) {
		return fmt.Errorf("POST %s: response body is not valid JSON", path)
	}
	return nil
}

// ---------------------------------------------------------------------------
// RESTClient – GET endpoints
// ---------------------------------------------------------------------------

// GetServerInfo returns general server information (version, name, GUID).
func (c *Client) GetServerInfo(ctx context.Context) (*ServerInfo, error) {
	var result ServerInfo
	if err := c.getInto(ctx, "/info", &result, c.maxBodyBytes); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPlayerList returns the list of connected players.
func (c *Client) GetPlayerList(ctx context.Context) (*PlayerList, error) {
	var result PlayerList
	if err := c.getInto(ctx, "/players", &result, c.maxBodyBytes); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetServerSettings returns the current server settings.
func (c *Client) GetServerSettings(ctx context.Context) (*ServerSettings, error) {
	var result ServerSettings
	if err := c.getInto(ctx, "/settings", &result, c.maxBodyBytes); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetServerMetrics returns the server performance metrics.
func (c *Client) GetServerMetrics(ctx context.Context) (*ServerMetrics, error) {
	var result ServerMetrics
	if err := c.getInto(ctx, "/metrics", &result, c.maxBodyBytes); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetGameData returns the world actor snapshot. Requires the server to run
// with -enable-gamedata-api. The response lists every actor in the world and
// can exceed the default 10 MiB cap on large servers; raise it with
// WithMaxResponseBytes when needed.
func (c *Client) GetGameData(ctx context.Context) (*GameData, error) {
	var result GameData
	if err := c.getInto(ctx, "/game-data", &result, c.maxBodyBytes); err != nil {
		return nil, err
	}
	return &result, nil
}

// ---------------------------------------------------------------------------
// RESTClient – POST endpoints
// ---------------------------------------------------------------------------

// MakeAnnouncement sends an announcement message to all players.
func (c *Client) MakeAnnouncement(ctx context.Context, message string) error {
	message, err := requiredTrimmed(message, "message")
	if err != nil {
		return err
	}
	return c.post(ctx, "/announce", map[string]string{"message": message})
}

// KickPlayer kicks a player from the server by user ID with a message.
func (c *Client) KickPlayer(ctx context.Context, userID, message string) error {
	return c.playerAction(ctx, "/kick", userID, message)
}

// BanPlayer bans a player from the server by user ID with a message.
func (c *Client) BanPlayer(ctx context.Context, userID, message string) error {
	return c.playerAction(ctx, "/ban", userID, message)
}

// UnbanPlayer unbans a player by user ID.
func (c *Client) UnbanPlayer(ctx context.Context, userID string) error {
	userID, err := requiredTrimmed(userID, "userid")
	if err != nil {
		return err
	}
	return c.post(ctx, "/unban", map[string]string{"userid": userID})
}

// SaveServerState saves the current world state on the server.
func (c *Client) SaveServerState(ctx context.Context) error {
	return c.post(ctx, "/save", nil)
}

// ShutdownServer shuts down the server after waitTime seconds with a message.
// The official documentation does not define the behavior of a waitTime of 0;
// observed server behavior is immediate shutdown. Use StopServer for force stop.
func (c *Client) ShutdownServer(ctx context.Context, waitTime int, message string) error {
	if waitTime < 0 {
		return errors.New("waittime must not be negative")
	}
	message = strings.TrimSpace(message)
	payload := map[string]any{"waittime": waitTime}
	if message != "" {
		payload["message"] = message
	}
	return c.post(ctx, "/shutdown", payload)
}

// playerAction kicks or bans a player by user ID, sending an optional message.
func (c *Client) playerAction(ctx context.Context, path, userID, message string) error {
	userID, err := requiredTrimmed(userID, "userid")
	if err != nil {
		return err
	}
	return c.post(ctx, path, playerPayload(userID, strings.TrimSpace(message)))
}

// playerPayload builds a kick/ban payload, omitting the optional message
// when empty.
func playerPayload(userID, message string) map[string]string {
	payload := map[string]string{"userid": userID}
	if message != "" {
		payload["message"] = message
	}
	return payload
}

// requiredTrimmed trims whitespace from value and fails when the result is
// empty.
func requiredTrimmed(value, name string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

// StopServer stops the server immediately.
func (c *Client) StopServer(ctx context.Context) error {
	return c.post(ctx, "/stop", nil)
}
