package palrest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient_NormalizesBaseURL(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	if client == nil {
		t.Fatal("client should not be nil")
	}

	if client.baseURL != "http://127.0.0.1:17999" {
		t.Fatalf("unexpected baseURL: %s", client.baseURL)
	}
}

// TestNewClient_CaseInsensitiveScheme pins that url.Parse normalizes the
// scheme to lowercase, so uppercase/mixed-case schemes are accepted and
// normalized instead of being rejected.
func TestNewClient_CaseInsensitiveScheme(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"HTTP://127.0.0.1:17999", "http://127.0.0.1:17999"},
		{"HtTpS://example.com:17999", "https://example.com:17999"},
	}
	for _, tt := range tests {
		client, err := NewClient(tt.raw, "secret")
		if err != nil {
			t.Fatalf("NewClient(%q) failed: %v", tt.raw, err)
		}
		if client.baseURL != tt.want {
			t.Fatalf("baseURL for %q, got %s, want %s", tt.raw, client.baseURL, tt.want)
		}
	}
}

func TestNewClient_DefaultPort(t *testing.T) {
	client, err := NewClient("127.0.0.1", "secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	if client.baseURL != "http://127.0.0.1:8212" {
		t.Fatalf("unexpected baseURL: %s", client.baseURL)
	}
}

func TestNewClient_IPv6Host(t *testing.T) {
	client, err := NewClient("[::1]:17999", "secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	if client.baseURL != "http://[::1]:17999" {
		t.Fatalf("unexpected baseURL: %s", client.baseURL)
	}
}

func TestNewClient_IPv6ZoneHost(t *testing.T) {
	client, err := NewClient("http://[fe80::1%25eth0]:17999", "secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	if client.baseURL != "http://[fe80::1%25eth0]:17999" {
		t.Fatalf("unexpected baseURL: %s", client.baseURL)
	}
}

func TestClient_IPv6ZoneHost_RequestURLParses(t *testing.T) {
	client, err := NewClient("http://[fe80::1%25eth0]:17999", "secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	target := client.buildURL("/info")
	if target != "http://[fe80::1%25eth0]:17999/v1/api/info" {
		t.Fatalf("unexpected target: %s", target)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, target, nil)
	if err != nil {
		t.Fatalf("request URL must be parseable: %v", err)
	}
	if req.URL.Hostname() != "fe80::1%eth0" {
		t.Fatalf("unexpected hostname: %s", req.URL.Hostname())
	}
}

func TestNewClient_RejectsUnsupportedBaseURLs(t *testing.T) {
	urls := []string{
		"http://127.0.0.1:8212/api",
		"http://127.0.0.1:8212/path/to/base",
		"http://127.0.0.1:8212?token=abc",
		"http://127.0.0.1:8212#section",
		"ftp://127.0.0.1:8212",
		"http://admin:secret@127.0.0.1:8212",
		"http://127.0.0.1:-1",
		"http://127.0.0.1:0",
		"http://127.0.0.1:65536",
		"http://exa_mple.com:8212",
		"http://-foo.com:8212",
		"http://foo-.com:8212",
		"http://a.-b.com:8212",
		"http://foo..bar.com:8212",
		"http://.:8212",
		"http://127.0.0.1:+8212",
		"http:/127.0.0.1:8212",
		"https:/127.0.0.1:8212",
		"http://" + strings.Repeat("a", 254) + ".com:8212",
	}
	for _, raw := range urls {
		_, err := NewClient(raw, "secret")
		if err == nil {
			t.Fatalf("expected error for base URL %q", raw)
		}
	}
}

func TestNewClient_RedactsUserinfoInErrors(t *testing.T) {
	for _, raw := range []string{
		"http://admin:secret@127.0.0.1:8212",
		"http://admin:p@ss@127.0.0.1:8212",
	} {
		_, err := NewClient(raw, "secret")
		if err == nil {
			t.Fatalf("expected error for userinfo in base URL %q", raw)
		}
		if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "p@ss") {
			t.Fatalf("credentials leaked in error message: %v", err)
		}
		if !strings.Contains(err.Error(), "xxxxx") {
			t.Fatalf("expected redacted userinfo in error message: %v", err)
		}
	}
}

func TestNewClient_MalformedSchemeDoesNotLeakCredentials(t *testing.T) {
	for _, raw := range []string{
		"http:/admin:secret@127.0.0.1:8212",
		"https:/user:p@ss@example.com:17999",
	} {
		_, err := NewClient(raw, "secret")
		if err == nil {
			t.Fatalf("expected error for malformed base URL %q", raw)
		}
		if strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "p@ss") {
			t.Fatalf("credentials leaked in error message: %v", err)
		}
	}
}

func TestNewClient_UnbracketedIPv6Normalized(t *testing.T) {
	client, err := NewClient("::1:2222", "secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	if client.baseURL != "http://[::1]:2222" {
		t.Fatalf("unexpected baseURL: %s", client.baseURL)
	}
}

func TestNewClient_UnbracketedIPv6WithoutPortRejected(t *testing.T) {
	if _, err := NewClient("::1", "secret"); err == nil {
		t.Fatal("expected error for unbracketed IPv6 without a port")
	}
}

func TestNewClient_TrimsWhitespaceBaseURL(t *testing.T) {
	client, err := NewClient("  127.0.0.1:17999  ", "secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	if client.baseURL != "http://127.0.0.1:17999" {
		t.Fatalf("unexpected baseURL: %s", client.baseURL)
	}
}

func TestNewClient_TrailingDotHost(t *testing.T) {
	client, err := NewClient("example.com.:17999", "secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	if client.baseURL != "http://example.com.:17999" {
		t.Fatalf("unexpected baseURL: %s", client.baseURL)
	}
}

func TestNewClient_MaxLengthFQDN(t *testing.T) {
	host253 := strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 61)
	if len(host253) != 253 {
		t.Fatalf("test fixture has %d chars", len(host253))
	}

	if _, err := NewClient(host253+":17999", "secret"); err != nil {
		t.Fatalf("253-char FQDN should be accepted: %v", err)
	}
	if _, err := NewClient(host253+".:17999", "secret"); err != nil {
		t.Fatalf("253-char FQDN with trailing dot should be accepted: %v", err)
	}

	host254 := strings.Repeat("a", 63) + "." + strings.Repeat("b", 63) + "." + strings.Repeat("c", 63) + "." + strings.Repeat("d", 62)
	if len(host254) != 254 {
		t.Fatalf("test fixture has %d chars", len(host254))
	}

	if _, err := NewClient(host254+":17999", "secret"); err == nil {
		t.Fatal("254-char FQDN should be rejected")
	}
	if _, err := NewClient(host254+".:17999", "secret"); err == nil {
		t.Fatal("255-char FQDN with trailing dot should be rejected")
	}
}

func TestNewClient_LabelLongerThan63CharsRejected(t *testing.T) {
	if _, err := NewClient(strings.Repeat("a", 64)+".com:17999", "secret"); err == nil {
		t.Fatal("expected error for a DNS label longer than 63 characters")
	}
}

func TestNewClient_PortWithLeadingDigitAtoiError(t *testing.T) {
	if _, err := NewClient("http://127.0.0.1:12x", "secret"); err == nil {
		t.Fatal("expected error for a non-numeric port starting with a digit")
	}
}

func TestNewClient_RequiresBaseURL(t *testing.T) {
	for _, raw := range []string{"", "   "} {
		_, err := NewClient(raw, "secret")
		if err == nil {
			t.Fatalf("expected error for base URL %q", raw)
		}
		if !strings.Contains(err.Error(), "base URL is required") {
			t.Fatalf("unexpected error for base URL %q: %v", raw, err)
		}
	}
}

func TestNewClient_LeadingZeroPort(t *testing.T) {
	client, err := NewClient("127.0.0.1:08212", "secret")
	if err != nil {
		t.Fatalf("unexpected error creating client: %v", err)
	}

	if client.baseURL != "http://127.0.0.1:8212" {
		t.Fatalf("unexpected baseURL: %s", client.baseURL)
	}
}

func TestNewClient_RejectsInvalidPort(t *testing.T) {
	_, err := NewClient("127.0.0.1:abc", "secret")
	if err == nil {
		t.Fatal("expected error for non-numeric port")
	}
}

func TestNewClient_RequiresPassword(t *testing.T) {
	_, err := NewClient("127.0.0.1:17999", "")
	if err == nil {
		t.Fatal("expected error when password is empty")
	}
}

func TestClient_GetServerInfo_UsesBasicAuth(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/info", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth header")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if user != "admin" || pass != "secret" {
			t.Errorf("unexpected credentials: %s/%s", user, pass)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ServerInfo{
			Version:     "1.0.0",
			ServerName:  "Test Server",
			Description: "desc",
			WorldGUID:   "guid-123",
		})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	info, err := client.GetServerInfo(context.Background())
	if err != nil {
		t.Fatalf("GetServerInfo failed: %v", err)
	}
	if info.ServerName != "Test Server" || info.WorldGUID != "guid-123" {
		t.Fatalf("unexpected response: %+v", info)
	}
}

func TestClient_ReturnsAPIError(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte(`{"error":"temporarily unavailable"}`))
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	_, err = client.GetServerMetrics(context.Background())
	if err == nil {
		t.Fatal("expected error for status >= 400")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("unexpected error type: %T", err)
	}
	if apiErr.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("unexpected status: %d", apiErr.StatusCode)
	}
	if apiErr.Path != "/metrics" {
		t.Fatalf("unexpected path: %s", apiErr.Path)
	}
}

func TestClient_NonJSONResponse(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/info", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("hello world"))
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	_, err = client.GetServerInfo(context.Background())
	if err == nil {
		t.Fatal("expected error decoding non-JSON response")
	}
	if !strings.Contains(err.Error(), `content-type "text/plain"`) {
		t.Fatalf("expected content-type in error, got: %v", err)
	}
}

func TestClient_EmptyBodyResponse(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/info", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	_, err = client.GetServerInfo(context.Background())
	if err == nil {
		t.Fatal("expected error for empty GET response body")
	}
}

func TestClient_NullBodyResponse(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/info", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("null"))
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	_, err = client.GetServerInfo(context.Background())
	if err == nil {
		t.Fatal("expected error for null GET response body")
	}
	if !strings.Contains(err.Error(), "null response body") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Post_RejectsEmptyBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/save", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if err := client.SaveServerState(context.Background()); err == nil {
		t.Fatal("expected error for empty POST success body")
	}
}

func TestClient_Post_RejectsWhitespaceBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/save", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("\n  \t\n"))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if err := client.SaveServerState(context.Background()); err == nil {
		t.Fatal("expected error for whitespace-only POST success body")
	}
}

func TestClient_Post_RejectsHTMLSuccess(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/save", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html>error page</html>"))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	err = client.SaveServerState(context.Background())
	if err == nil {
		t.Fatal("expected error for HTML success response")
	}
	if !strings.Contains(err.Error(), `unexpected content-type "text/html"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Post_RejectsHTMLContentTypeWithEmptyBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/save", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	err = client.SaveServerState(context.Background())
	if err == nil {
		t.Fatal("expected error for empty success body with non-JSON content type")
	}
	if !strings.Contains(err.Error(), `unexpected content-type "text/html"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Post_RejectsNonJSONBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/stop", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("ok"))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	err = client.StopServer(context.Background())
	if err == nil {
		t.Fatal("expected error for non-matching success body")
	}
}

func TestClient_Post_RejectsJSONBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/announce", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if err := client.MakeAnnouncement(context.Background(), "hello"); err == nil {
		t.Fatal("expected error for JSON POST success body")
	}
}

func TestClient_Post_AcceptsTextPlainResponse(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/announce", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain;charset=utf-8")
		_, _ = w.Write([]byte(announceSuccessText + "\n"))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if err := client.MakeAnnouncement(context.Background(), "hello"); err != nil {
		t.Fatalf("MakeAnnouncement should accept the documented text/plain response: %v", err)
	}
}

func TestClient_Post_AcceptsUppercaseContentType(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/announce", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "Text/Plain; charset=UTF-8")
		_, _ = w.Write([]byte(announceSuccessText))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if err := client.MakeAnnouncement(context.Background(), "hello"); err != nil {
		t.Fatalf("MakeAnnouncement should accept case-insensitive text/plain content types: %v", err)
	}
}

func TestClient_Post_RejectsWrongPlainTextBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/stop", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = w.Write([]byte("ok"))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	err = client.StopServer(context.Background())
	if err == nil {
		t.Fatal("expected error for non-matching plain text body")
	}
	if !strings.Contains(err.Error(), "unexpected response body") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Post_RejectsDivergentPlainText(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		body   string
		action func(ctx context.Context, c *Client) error
	}{
		{
			name: "wrong text",
			path: "/v1/api/announce",
			body: "Wrong message.",
			action: func(ctx context.Context, c *Client) error {
				return c.MakeAnnouncement(ctx, "hello")
			},
		},
		{
			name: "another endpoint's text",
			path: "/v1/api/shutdown",
			body: announceSuccessText,
			action: func(ctx context.Context, c *Client) error {
				return c.ShutdownServer(ctx, 5, "bye")
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := http.NewServeMux()
			handler.HandleFunc(tt.path, func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/plain")
				_, _ = w.Write([]byte(tt.body))
			})
			srv := httptest.NewServer(handler)
			defer srv.Close()

			client, err := NewClient(srv.URL, "secret")
			if err != nil {
				t.Fatalf("error creating client: %v", err)
			}
			if err := tt.action(context.Background(), client); err == nil {
				t.Fatal("expected error for non-matching plain text body")
			}
		})
	}
}

func TestClient_Post_RejectsJSONContentTypeWithDocumentedText(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/save", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(saveSuccessText))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	err = client.SaveServerState(context.Background())
	if err == nil {
		t.Fatal("expected error for JSON content-type even with the documented text")
	}
	if !strings.Contains(err.Error(), `unexpected content-type "application/json"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_Post_AcceptsDocumentedTextWithoutContentType(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/stop", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(stopSuccessText))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if err := client.StopServer(context.Background()); err != nil {
		t.Fatalf("StopServer should accept the documented text without a content-type: %v", err)
	}
}

func TestClient_Post_RejectsOversizedSuccessBody(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/save", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":"` + strings.Repeat("x", postResponseLimitBytes+64) + `"}`))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if err := client.SaveServerState(context.Background()); err == nil {
		t.Fatal("expected error for oversized POST success body")
	}
}

func TestClient_TransportErrorWrapped(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret", WithHTTPClient(&http.Client{
		Transport: failingTransport{},
	}))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	_, err = client.GetServerInfo(context.Background())
	if err == nil {
		t.Fatal("expected transport error")
	}
	if !strings.Contains(err.Error(), "GET http://127.0.0.1:17999/v1/api/info") {
		t.Fatalf("expected full URL in error, got: %v", err)
	}
}

func TestClient_DoesNotFollowRedirects(t *testing.T) {
	var redirectTargetHit atomic.Int32
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/info", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/v1/api/info/", http.StatusTemporaryRedirect)
	})
	handler.HandleFunc("/v1/api/info/", func(w http.ResponseWriter, r *http.Request) {
		redirectTargetHit.Add(1)
		_ = json.NewEncoder(w).Encode(ServerInfo{ServerName: "ShouldNotReach"})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if _, err := client.GetServerInfo(context.Background()); err == nil {
		t.Fatal("expected error for redirect response")
	}
	if redirectTargetHit.Load() != 0 {
		t.Fatal("client followed a redirect")
	}
}

func TestClient_DoesNotFollowRedirects_OnWriteEndpoints(t *testing.T) {
	var saveHit atomic.Int32
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/save", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/v1/api/save/", http.StatusPermanentRedirect)
	})
	handler.HandleFunc("/v1/api/save/", func(w http.ResponseWriter, r *http.Request) {
		saveHit.Add(1)
		w.WriteHeader(http.StatusOK)
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if err := client.SaveServerState(context.Background()); err == nil {
		t.Fatal("expected error for redirect response")
	}
	if saveHit.Load() != 0 {
		t.Fatal("client followed a redirect")
	}
}

func TestClient_WithTimeoutIgnoresNonPositive(t *testing.T) {
	for _, d := range []time.Duration{0, -time.Second} {
		client, err := NewClient("127.0.0.1:17999", "secret", WithTimeout(d))
		if err != nil {
			t.Fatalf("error creating client: %v", err)
		}
		if client.client.Timeout != defaultTimeout {
			t.Fatalf("unexpected timeout for %v: %v", d, client.client.Timeout)
		}
	}
}

func TestClient_InternalTransportIgnoresProxy(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	transport, ok := client.client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("unexpected transport type: %T", client.client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("internal transport must not use environment proxies")
	}
}

type failingTransport struct{}

func (failingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection refused")
}

// Exercises the read endpoints with distinct response shapes and assertions.
func TestClient_ReadEndpoints(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/info", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ServerInfo{ServerName: "InfoServer"})
	})
	handler.HandleFunc("/v1/api/players", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(PlayerList{Players: []PlayerInfo{{
			Name: "PlayerOne", AccountName: "acct", PlayerID: "ABC123", UserID: "steam_1",
			IP: "127.0.0.1", Ping: 3.14, LocationX: 123.45, LocationY: 67.89,
			Level: 5, BuildingCount: 2,
		}}})
	})
	handler.HandleFunc("/v1/api/settings", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ServerSettings{
			Difficulty: "Normal", IsPvP: true, ServerPlayerMaxNum: 32, ServerName: "Srv",
		})
	})
	handler.HandleFunc("/v1/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ServerMetrics{ServerFPS: 60, CurrentPlayerNum: 10})
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	ctx := context.Background()

	info, err := client.GetServerInfo(ctx)
	if err != nil {
		t.Fatalf("GetServerInfo failed: %v", err)
	}
	if info.ServerName != "InfoServer" {
		t.Fatalf("unexpected info: %+v", info)
	}

	players, err := client.GetPlayerList(ctx)
	if err != nil {
		t.Fatalf("GetPlayerList failed: %v", err)
	}
	if len(players.Players) == 0 {
		t.Fatalf("empty player list: %+v", players)
	}
	p := players.Players[0]
	if p.Name != "PlayerOne" || p.UserID != "steam_1" || p.Ping != 3.14 || p.Level != 5 || p.BuildingCount != 2 {
		t.Fatalf("unexpected player: %+v", p)
	}

	settings, err := client.GetServerSettings(ctx)
	if err != nil {
		t.Fatalf("GetServerSettings failed: %v", err)
	}
	if settings.Difficulty != "Normal" || !settings.IsPvP || settings.ServerPlayerMaxNum != 32 {
		t.Fatalf("unexpected settings: %+v", settings)
	}

	metrics, err := client.GetServerMetrics(ctx)
	if err != nil {
		t.Fatalf("GetServerMetrics failed: %v", err)
	}
	if metrics.ServerFPS != 60 || metrics.CurrentPlayerNum != 10 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

// TestClient_GetServerSettings_FullSchema decodes the complete /settings
// response documented in the official REST API reference (v1.0.2) and
// rejects any field the client schema does not cover.
func TestClient_GetServerSettings_FullSchema(t *testing.T) {
	raw := `{
		"Difficulty": "None",
		"RandomizerType": "None",
		"RandomizerSeed": "",
		"bIsRandomizerPalLevelRandom": false,
		"DayTimeSpeedRate": 1.0,
		"NightTimeSpeedRate": 1.0,
		"ExpRate": 1.0,
		"PalCaptureRate": 1.0,
		"PalSpawnNumRate": 1.0,
		"PalDamageRateAttack": 1.0,
		"PalDamageRateDefense": 1.0,
		"PlayerDamageRateAttack": 1.0,
		"PlayerDamageRateDefense": 1.0,
		"PlayerStomachDecreaceRate": 1.0,
		"PlayerStaminaDecreaceRate": 1.0,
		"PlayerAutoHPRegeneRate": 1.0,
		"PlayerAutoHpRegeneRateInSleep": 1.0,
		"PalStomachDecreaceRate": 1.0,
		"PalStaminaDecreaceRate": 1.0,
		"PalAutoHPRegeneRate": 1.0,
		"PalAutoHpRegeneRateInSleep": 1.0,
		"BuildObjectHpRate": 1.0,
		"BuildObjectDamageRate": 1.0,
		"BuildObjectDeteriorationDamageRate": 1.0,
		"CollectionDropRate": 1.0,
		"CollectionObjectHpRate": 1.0,
		"CollectionObjectRespawnSpeedRate": 1.0,
		"EnemyDropItemRate": 1.0,
		"DeathPenalty": "Item",
		"bEnablePlayerToPlayerDamage": false,
		"bEnableFriendlyFire": false,
		"bEnableInvaderEnemy": true,
		"bActiveUNKO": false,
		"bEnableAimAssistPad": true,
		"bEnableAimAssistKeyboard": false,
		"DropItemMaxNum": 3000,
		"PhysicsActiveDropItemMaxNum": -1,
		"DropItemMaxNum_UNKO": 100,
		"BaseCampMaxNum": 128,
		"BaseCampWorkerMaxNum": 15,
		"DropItemAliveMaxHours": 1.0,
		"bAutoResetGuildNoOnlinePlayers": false,
		"AutoResetGuildTimeNoOnlinePlayers": 72.0,
		"GuildPlayerMaxNum": 20,
		"BaseCampMaxNumInGuild": 4,
		"PalEggDefaultHatchingTime": 1.0,
		"WorkSpeedRate": 1.0,
		"autoSaveSpan": 30,
		"bIsMultiplay": false,
		"bIsPvP": false,
		"bHardcore": false,
		"bPalLost": false,
		"bCharacterRecreateInHardcore": false,
		"bCanPickupOtherGuildDeathPenaltyDrop": false,
		"bEnableNonLoginPenalty": true,
		"bEnableFastTravel": true,
		"bEnableFastTravelOnlyBaseCamp": false,
		"bIsStartLocationSelectByMap": false,
		"bExistPlayerAfterLogout": false,
		"bEnableDefenseOtherGuildPlayer": false,
		"bInvisibleOtherGuildBaseCampAreaFX": false,
		"bBuildAreaLimit": false,
		"ItemWeightRate": 1.0,
		"CoopPlayerMaxNum": 4,
		"ServerPlayerMaxNum": 32,
		"ServerName": "Default Palworld Server",
		"ServerDescription": "",
		"bAllowClientMod": true,
		"PublicPort": 8211,
		"PublicIP": "",
		"RCONEnabled": false,
		"RCONPort": 25575,
		"Region": "",
		"bUseAuth": true,
		"BanListURL": "https://b.palworldgame.com/api/banlist.txt",
		"RESTAPIEnabled": false,
		"RESTAPIPort": 8212,
		"bShowPlayerList": false,
		"ChatPostLimitPerMinute": 30,
		"CrossplayPlatforms": ["Steam", "Xbox", "PS5", "Mac"],
		"bIsUseBackupSaveData": true,
		"LogFormatType": "Text",
		"bIsShowJoinLeftMessage": true,
		"SupplyDropSpan": 180,
		"EnablePredatorBossPal": true,
		"MaxBuildingLimitNum": 0,
		"ServerReplicatePawnCullDistance": 15000,
		"bAllowGlobalPalboxExport": true,
		"bAllowGlobalPalboxImport": false,
		"EquipmentDurabilityDamageRate": 1.0,
		"ItemContainerForceMarkDirtyInterval": 1.0,
		"PlayerDataPalStorageUpdateCheckTickInterval": 1.0,
		"ItemCorruptionMultiplier": 1.0,
		"MonsterFarmActionSpeedRate": 1.0,
		"DenyTechnologyList": [],
		"GuildRejoinCooldownMinutes": 0,
		"AutoTransferMasterCheckIntervalSeconds": 3600,
		"AutoTransferMasterThresholdDays": 14,
		"MaxGuildsPerFrame": 10,
		"BlockRespawnTime": 5,
		"RespawnPenaltyDurationThreshold": 0,
		"RespawnPenaltyTimeScale": 2,
		"bDisplayPvPItemNumOnWorldMap_BaseCamp": false,
		"bDisplayPvPItemNumOnWorldMap_Player": false,
		"AdditionalDropItemWhenPlayerKillingInPvPMode": "PlayerDropItem",
		"AdditionalDropItemNumWhenPlayerKillingInPvPMode": 1,
		"bAdditionalDropItemWhenPlayerKillingInPvPMode": false,
		"bEnableVoiceChat": false,
		"VoiceChatMaxVolumeDistance": 3000,
		"VoiceChatZeroVolumeDistance": 15000,
		"bAllowEnhanceStat_Health": true,
		"bAllowEnhanceStat_Attack": true,
		"bAllowEnhanceStat_Stamina": true,
		"bAllowEnhanceStat_Weight": true,
		"bAllowEnhanceStat_WorkSpeed": true,
		"bEnableBuildingPlayerUIdDisplay": false,
		"BuildingNameDisplayCacheTTLSeconds": 60,
		"bAllowEnemyCampSpawnNearBaseCamp": false,
		"AllowConnectPlatform": "Steam"
	}`

	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/settings", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(raw))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	settings, err := client.GetServerSettings(context.Background())
	if err != nil {
		t.Fatalf("GetServerSettings failed: %v", err)
	}

	decodeStrict(t, raw, &ServerSettings{})

	if settings.Difficulty != "None" || settings.DeathPenalty != "Item" ||
		settings.PalEggDefaultHatchingTime != 1.0 || settings.BaseCampMaxNumInGuild != 4 ||
		settings.BuildObjectHpRate != 1.0 || settings.ItemWeightRate != 1.0 ||
		settings.AutoSaveSpan != 30 || settings.ServerName != "Default Palworld Server" ||
		settings.ServerDescription != "" || settings.PublicIP != "" || settings.Region != "" ||
		settings.BanListURL != "https://b.palworldgame.com/api/banlist.txt" ||
		settings.RESTAPIEnabled || settings.RCONEnabled || settings.ShowPlayerList || settings.IsMultiplay ||
		!settings.AllowClientMod || settings.MaxBuildingLimitNum != 0 ||
		settings.SupplyDropSpan != 180 || settings.ServerReplicatePawnCullDistance != 15000 ||
		settings.ItemCorruptionMultiplier != 1.0 || !settings.AllowGlobalPalboxExport ||
		!settings.EnablePredatorBossPal || len(settings.CrossplayPlatforms) != 4 ||
		!settings.AllowEnhanceStat_Health || settings.AllowEnemyCampSpawnNearBaseCamp ||
		!settings.IsUseBackupSaveData || settings.AllowConnectPlatform != "Steam" {
		t.Fatalf("unexpected settings: %+v", settings)
	}
}

// decodeStrict verifies that every field of raw is covered by the client schema.
func decodeStrict(t *testing.T, raw string, out any) {
	t.Helper()

	dec := json.NewDecoder(strings.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(out); err != nil {
		t.Fatalf("client schema does not cover a documented field: %v", err)
	}
}

// testClientServing returns a client backed by a test server that serves raw
// on path for every GET request.
func testClientServing(t *testing.T, path, raw string, opts ...Option) *Client {
	t.Helper()

	handler := http.NewServeMux()
	handler.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("unexpected method: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		_, _ = w.Write([]byte(raw))
	})
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	client, err := NewClient(srv.URL, "secret", opts...)
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}
	return client
}

func TestClient_GetServerInfo_FullSchema(t *testing.T) {
	raw := `{
		"version": "v0.1.5.0",
		"servername": "Palworld example Server",
		"description": "This is a Palworld server.",
		"worldguid": "A7E97BAA767DB9029EF013BB71E993A0"
	}`

	client := testClientServing(t, "/v1/api/info", raw)
	info, err := client.GetServerInfo(context.Background())
	if err != nil {
		t.Fatalf("GetServerInfo failed: %v", err)
	}

	decodeStrict(t, raw, &ServerInfo{})

	if info.Version != "v0.1.5.0" || info.ServerName != "Palworld example Server" ||
		info.Description != "This is a Palworld server." || info.WorldGUID != "A7E97BAA767DB9029EF013BB71E993A0" {
		t.Fatalf("unexpected info: %+v", info)
	}
}

func TestClient_GetPlayerList_FullSchema(t *testing.T) {
	raw := `{
		"players": [
			{
				"name": "PalUser",
				"accountName": "paluser",
				"playerId": "AFAFD830000000000000000000000000",
				"userId": "steam_00000000000000000",
				"ip": "127.0.0.1",
				"ping": 3.14,
				"location_x": 123.45,
				"location_y": 67.89,
				"level": 1,
				"building_count": 119
			}
		]
	}`

	client := testClientServing(t, "/v1/api/players", raw)
	players, err := client.GetPlayerList(context.Background())
	if err != nil {
		t.Fatalf("GetPlayerList failed: %v", err)
	}

	decodeStrict(t, raw, &PlayerList{})

	p := players.Players[0]
	if p.Name != "PalUser" || p.AccountName != "paluser" ||
		p.PlayerID != "AFAFD830000000000000000000000000" || p.UserID != "steam_00000000000000000" ||
		p.IP != "127.0.0.1" || p.Ping != 3.14 || p.LocationX != 123.45 || p.LocationY != 67.89 ||
		p.Level != 1 || p.BuildingCount != 119 {
		t.Fatalf("unexpected player: %+v", p)
	}
}

func TestClient_GetServerMetrics_FullSchema(t *testing.T) {
	raw := `{
		"serverfps": 57,
		"currentplayernum": 10,
		"serverframetime": 16.7671,
		"maxplayernum": 32,
		"uptime": 3600,
		"basecampnum": 32,
		"days": 1
	}`

	client := testClientServing(t, "/v1/api/metrics", raw)
	metrics, err := client.GetServerMetrics(context.Background())
	if err != nil {
		t.Fatalf("GetServerMetrics failed: %v", err)
	}

	decodeStrict(t, raw, &ServerMetrics{})

	if metrics.ServerFPS != 57 || metrics.CurrentPlayerNum != 10 || metrics.ServerFrameTime != 16.7671 ||
		metrics.MaxPlayerNum != 32 || metrics.Uptime != 3600 || metrics.BaseCampNum != 32 || metrics.Days != 1 {
		t.Fatalf("unexpected metrics: %+v", metrics)
	}
}

const (
	gameDataCharacter = `{
		"Type": "Character",
		"InstanceID": "char-1",
		"UnitType": "Player",
		"NickName": "PalUser",
		"TrainerInstanceID": "char-0",
		"TrainerNickName": "Trainer",
		"TrainerClass": "BP_PlayerCharacter",
		"userid": "steam_1",
		"ip": "127.0.0.1",
		"level": 42,
		"HP": 100,
		"MaxHP": 120,
		"GuildID": "guild-1",
		"GuildName": "TestGuild",
		"Class": "BP_PlayerCharacter",
		"Action": "Idle",
		"AI_Action": "None",
		"LocationX": 1.5,
		"LocationY": 2.5,
		"LocationZ": 3.5,
		"RotationX": 0.1,
		"RotationY": 0.2,
		"RotationZ": 0.3,
		"Stage": "Stage1",
		"IsActive": "true"
	}`
	gameDataPalBox = `{
		"Type": "PalBox",
		"GuildID": "guild-2",
		"GuildName": "TestGuild",
		"Class": "BP_PalBox",
		"LocationX": 4.2,
		"LocationY": 5.2,
		"LocationZ": 6.2
	}`
	gameDataSnapshot = `{
		"Time": "2026-06-17 13:00:40",
		"FPS": 91.71,
		"AverageFPS": 33.78,
		"ActorData": [` + gameDataCharacter + `, ` + gameDataPalBox + `]
	}`
)

// TestClient_GetGameData_FullSchema decodes the /game-data snapshot documented
// in the official REST API reference (v1.0.2). The strict decode of the
// snapshot pins the top-level schema; actor-inner field coverage is pinned
// against the documented fixtures via the two direct strict decodes below.
// The runtime decode path (Actor.UnmarshalJSON) deliberately tolerates unknown
// actor fields and kinds for forward compatibility, so unknown fields inside
// an actor must not be rejected here.
func TestClient_GetGameData_FullSchema(t *testing.T) {
	client := testClientServing(t, "/v1/api/game-data", gameDataSnapshot)
	data, err := client.GetGameData(context.Background())
	if err != nil {
		t.Fatalf("GetGameData failed: %v", err)
	}

	decodeStrict(t, gameDataSnapshot, &GameData{})
	decodeStrict(t, gameDataCharacter, &CharacterActor{})
	decodeStrict(t, gameDataPalBox, &PalBoxActor{})

	if len(data.ActorData) != 2 || data.ActorData[0].Character == nil || data.ActorData[1].PalBox == nil {
		t.Fatalf("unexpected snapshot: %+v", data)
	}
	c := data.ActorData[0].Character
	if c.InstanceID != "char-1" || c.UnitType != "Player" || c.TrainerInstanceID != "char-0" ||
		c.TrainerNickName != "Trainer" || c.TrainerClass != "BP_PlayerCharacter" || c.GuildID != "guild-1" ||
		c.Class != "BP_PlayerCharacter" || c.Action != "Idle" || c.AIAction != "None" ||
		c.RotationY != 0.2 || c.IsActive != "true" {
		t.Fatalf("unexpected character actor: %+v", c)
	}
}

func TestClient_GameDataRespectsGeneralCap(t *testing.T) {
	nick := strings.Repeat("x", 2048)
	raw := `{"Time":"2026-06-17 13:00:40","FPS":60,"AverageFPS":30,"ActorData":[{"Type":"Character","NickName":"` + nick + `"}]}`

	client := testClientServing(t, "/v1/api/game-data", raw, WithMaxResponseBytes(1024))
	if _, err := client.GetGameData(context.Background()); err == nil {
		t.Fatal("expected the general cap to apply to /game-data")
	}

	raised := testClientServing(t, "/v1/api/game-data", raw, WithMaxResponseBytes(4096))
	data, err := raised.GetGameData(context.Background())
	if err != nil {
		t.Fatalf("GetGameData should succeed with a raised cap: %v", err)
	}
	if len(data.ActorData) != 1 || data.ActorData[0].Character == nil {
		t.Fatalf("unexpected snapshot: %+v", data)
	}
}

func TestClient_ErrorBodyOverLimitStillReturnsAPIError(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(strings.Repeat("x", 4096)))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret", WithMaxResponseBytes(1024))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	_, err = client.GetServerMetrics(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got: %v", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", apiErr.StatusCode)
	}
	if body, ok := apiErr.ResponseBody.(string); !ok || len(body) > maxErrorBodyBytes {
		t.Fatalf("unexpected error body: %T", apiErr.ResponseBody)
	}
}

func TestClient_GetGameData_NullActorKept(t *testing.T) {
	data := fetchGameData(t, `{
		"Time": "2026-06-17 13:00:40",
		"FPS": 60,
		"AverageFPS": 30,
		"ActorData": [null, {"Type": "Character", "InstanceID": "char-1"}]
	}`)

	if len(data.ActorData) != 2 {
		t.Fatalf("unexpected snapshot: %+v", data)
	}
	if data.ActorData[0].Character != nil || data.ActorData[0].PalBox != nil || data.ActorData[0].Type != "" {
		t.Fatalf("null actor should stay zeroed: %+v", data.ActorData[0])
	}
	if data.ActorData[1].Character == nil {
		t.Fatalf("character actor missing: %+v", data.ActorData[1])
	}
}

func TestClient_GetGameData_MissingType(t *testing.T) {
	client := testClientServing(t, "/v1/api/game-data", `{
		"Time": "2026-06-17 13:00:40",
		"FPS": 60,
		"AverageFPS": 30,
		"ActorData": [{"InstanceID": "char-1"}]
	}`)

	if _, err := client.GetGameData(context.Background()); err == nil {
		t.Fatal("expected error for actor without Type discriminator")
	}
}

func TestClient_WriteEndpoints_TrimsValues(t *testing.T) {
	payloads := map[string]string{}
	var mu sync.Mutex
	handler := http.NewServeMux()
	for _, p := range []string{"/v1/api/announce", "/v1/api/kick", "/v1/api/ban", "/v1/api/shutdown"} {
		handler.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			bytes, _ := io.ReadAll(r.Body)
			mu.Lock()
			payloads[r.URL.Path] = string(bytes)
			mu.Unlock()
			_, _ = w.Write([]byte(postSuccessTexts[r.URL.Path]))
		})
	}
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	ctx := context.Background()
	if err := client.MakeAnnouncement(ctx, "  hello  "); err != nil {
		t.Fatalf("MakeAnnouncement failed: %v", err)
	}
	if err := client.KickPlayer(ctx, "  p1  ", "  afk  "); err != nil {
		t.Fatalf("KickPlayer failed: %v", err)
	}
	if err := client.BanPlayer(ctx, " p2 ", " cheat "); err != nil {
		t.Fatalf("BanPlayer failed: %v", err)
	}
	if err := client.ShutdownServer(ctx, 5, "  bye  "); err != nil {
		t.Fatalf("ShutdownServer failed: %v", err)
	}

	want := map[string]string{
		"/v1/api/announce": `{"message":"hello"}`,
		"/v1/api/kick":     `{"message":"afk","userid":"p1"}`,
		"/v1/api/ban":      `{"message":"cheat","userid":"p2"}`,
		"/v1/api/shutdown": `{"message":"bye","waittime":5}`,
	}
	for path, expected := range want {
		mu.Lock()
		got := payloads[path]
		mu.Unlock()
		if got != expected {
			t.Fatalf("payload on %s = %s, want %s", path, got, expected)
		}
	}
}

func TestClient_ShutdownServer_ZeroWaitTimeAllowed(t *testing.T) {
	var mu sync.Mutex
	payload := ""
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		bytes, _ := io.ReadAll(r.Body)
		mu.Lock()
		payload = string(bytes)
		mu.Unlock()
		_, _ = w.Write([]byte(shutdownSuccessText))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if err := client.ShutdownServer(context.Background(), 0, ""); err != nil {
		t.Fatalf("ShutdownServer with waittime 0 should succeed: %v", err)
	}
	mu.Lock()
	got := payload
	mu.Unlock()
	if got != `{"waittime":0}` {
		t.Fatalf("unexpected payload: %s", got)
	}
}

// fetchGameData builds a test client serving raw JSON for /v1/api/game-data
// and returns the decoded snapshot.
func fetchGameData(t *testing.T, raw string) *GameData {
	t.Helper()

	client := testClientServing(t, "/v1/api/game-data", raw)
	data, err := client.GetGameData(context.Background())
	if err != nil {
		t.Fatalf("GetGameData failed: %v", err)
	}
	return data
}

func TestClient_GetGameData(t *testing.T) {
	data := fetchGameData(t, gameDataSnapshot)

	want := &GameData{
		Time:       "2026-06-17 13:00:40",
		FPS:        91.71,
		AverageFPS: 33.78,
		ActorData: []Actor{
			{
				Type: "Character",
				Character: &CharacterActor{
					Type: "Character", InstanceID: "char-1", UnitType: "Player", NickName: "PalUser",
					TrainerInstanceID: "char-0", TrainerNickName: "Trainer", TrainerClass: "BP_PlayerCharacter",
					UserID: "steam_1", IP: "127.0.0.1", Level: 42, HP: 100, MaxHP: 120,
					GuildID: "guild-1", GuildName: "TestGuild", Class: "BP_PlayerCharacter",
					Action: "Idle", AIAction: "None",
					LocationX: 1.5, LocationY: 2.5, LocationZ: 3.5,
					RotationX: 0.1, RotationY: 0.2, RotationZ: 0.3,
					Stage: "Stage1", IsActive: "true",
				},
			},
			{
				Type: "PalBox",
				PalBox: &PalBoxActor{
					Type: "PalBox", GuildID: "guild-2", GuildName: "TestGuild",
					Class: "BP_PalBox", LocationX: 4.2, LocationY: 5.2, LocationZ: 6.2,
				},
			},
		},
	}
	if !reflect.DeepEqual(data, want) {
		t.Fatalf("unexpected snapshot:\n got %+v\nwant %+v", data, want)
	}
}

func TestClient_GetGameData_UnknownActorKind(t *testing.T) {
	data := fetchGameData(t, `{
		"Time": "2026-06-17 13:00:40",
		"FPS": 60,
		"AverageFPS": 30,
		"ActorData": [{"Type": "FutureKind", "Whatever": 1}]
	}`)

	actor := data.ActorData[0]
	if actor.Type != "FutureKind" || actor.Character != nil || actor.PalBox != nil {
		t.Fatalf("unexpected unknown actor: %+v", actor)
	}
}

func TestClient_GetGameData_UnknownActorFieldTolerated(t *testing.T) {
	data := fetchGameData(t, `{
		"Time": "2026-06-17 13:00:40",
		"FPS": 60,
		"AverageFPS": 30,
		"ActorData": [{"Type": "Character", "InstanceID": "char-1", "FutureField": 123}]
	}`)

	if len(data.ActorData) != 1 || data.ActorData[0].Character == nil {
		t.Fatalf("unexpected snapshot: %+v", data)
	}
	if c := data.ActorData[0].Character; c.InstanceID != "char-1" {
		t.Fatalf("unexpected character actor: %+v", c)
	}
}

func TestClient_Unauthorized(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/players", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid credentials"}`))
	})

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "wrong-password")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	_, err = client.GetPlayerList(context.Background())
	if err == nil {
		t.Fatal("expected error for status 401")
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("unexpected error type: %T", err)
	}
	if apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected status: %d", apiErr.StatusCode)
	}
}

// writeEndpointCall records a single request observed by the write endpoint test.
type writeEndpointCall struct {
	path   string
	method string
	body   string
}

// postSuccessTexts maps each POST endpoint path to its documented plain-text
// success confirmation.
var postSuccessTexts = map[string]string{
	"/v1/api/announce": announceSuccessText,
	"/v1/api/kick":     kickSuccessText,
	"/v1/api/ban":      banSuccessText,
	"/v1/api/unban":    unbanSuccessText,
	"/v1/api/save":     saveSuccessText,
	"/v1/api/shutdown": shutdownSuccessText,
	"/v1/api/stop":     stopSuccessText,
}

func TestClient_WriteEndpoints(t *testing.T) {
	paths := []string{
		"/v1/api/announce",
		"/v1/api/kick",
		"/v1/api/ban",
		"/v1/api/unban",
		"/v1/api/save",
		"/v1/api/shutdown",
		"/v1/api/stop",
	}

	calls := make([]writeEndpointCall, 0, len(paths))
	var mu sync.Mutex
	handler := http.NewServeMux()
	for _, p := range paths {
		handler.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			body := ""
			if r.Body != nil {
				bytes, _ := io.ReadAll(r.Body)
				body = string(bytes)
			}
			mu.Lock()
			calls = append(calls, writeEndpointCall{path: r.URL.Path, method: r.Method, body: body})
			mu.Unlock()
			w.Header().Set("Content-Type", "text/plain;charset=utf-8")
			_, _ = w.Write([]byte(postSuccessTexts[r.URL.Path]))
		})
	}

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	ctx := context.Background()
	actions := []struct {
		name string
		do   func() error
	}{
		{"MakeAnnouncement", func() error { return client.MakeAnnouncement(ctx, "hello") }},
		{"KickPlayer", func() error { return client.KickPlayer(ctx, "p1", "afk") }},
		{"BanPlayer", func() error { return client.BanPlayer(ctx, "p3", "cheat") }},
		{"UnbanPlayer", func() error { return client.UnbanPlayer(ctx, "p3") }},
		{"SaveServerState", func() error { return client.SaveServerState(ctx) }},
		{"ShutdownServer", func() error { return client.ShutdownServer(ctx, 10, "bye") }},
		{"StopServer", func() error { return client.StopServer(ctx) }},
	}
	for _, a := range actions {
		if err := a.do(); err != nil {
			t.Fatalf("%s failed: %v", a.name, err)
		}
	}

	mu.Lock()
	snapshot := make([]writeEndpointCall, len(calls))
	copy(snapshot, calls)
	mu.Unlock()
	assertCalls(t, snapshot, paths)
}

// assertCalls verifies that each expected path was called exactly once with
// method POST and that the payloads carry the documented field names.
func assertCalls(t *testing.T, calls []writeEndpointCall, paths []string) {
	t.Helper()

	if len(calls) != len(paths) {
		t.Fatalf("unexpected number of calls: %d", len(calls))
	}

	seen := map[string]int{}
	for _, c := range calls {
		if c.method != http.MethodPost {
			t.Fatalf("unexpected method on %s: %s", c.path, c.method)
		}
		seen[c.path]++
	}

	for _, path := range paths {
		if seen[path] != 1 {
			t.Fatalf("endpoint %s should have 1 call, had %d", path, seen[path])
		}
	}

	required := map[string][]string{
		"/v1/api/announce": {"message"},
		"/v1/api/kick":     {"userid", "message"},
		"/v1/api/ban":      {"userid", "message"},
		"/v1/api/unban":    {"userid"},
		"/v1/api/shutdown": {"waittime", "message"},
	}
	for _, c := range calls {
		if c.path == "/v1/api/shutdown" {
			assertShutdownPayload(t, c.body)
		}
		for _, field := range required[c.path] {
			if !strings.Contains(c.body, field) {
				t.Fatalf("payload on %s missing field %q: %s", c.path, field, c.body)
			}
		}
	}
}

// assertShutdownPayload verifies the exact JSON payload sent to /v1/api/shutdown.
func assertShutdownPayload(t *testing.T, body string) {
	t.Helper()
	var payload map[string]any
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("invalid shutdown payload %q: %v", body, err)
	}
	if payload["waittime"] != float64(10) || payload["message"] != "bye" {
		t.Fatalf("unexpected shutdown payload: %s", body)
	}
}

func TestClient_WriteEndpoints_ValidateRequiredFields(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	ctx := context.Background()
	actions := []struct {
		name string
		do   func() error
	}{
		{"MakeAnnouncement", func() error { return client.MakeAnnouncement(ctx, "") }},
		{"KickPlayer", func() error { return client.KickPlayer(ctx, "", "afk") }},
		{"BanPlayer", func() error { return client.BanPlayer(ctx, " ", "cheat") }},
		{"UnbanPlayer", func() error { return client.UnbanPlayer(ctx, "") }},
	}
	for _, a := range actions {
		if err := a.do(); err == nil {
			t.Fatalf("%s should fail validation", a.name)
		}
	}
}

func TestClient_WriteEndpoints_OmitsEmptyMessage(t *testing.T) {
	payloads := map[string]string{}
	var mu sync.Mutex
	handler := http.NewServeMux()
	for _, p := range []string{"/v1/api/kick", "/v1/api/ban", "/v1/api/shutdown"} {
		handler.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			bytes, _ := io.ReadAll(r.Body)
			mu.Lock()
			payloads[r.URL.Path] = string(bytes)
			mu.Unlock()
			_, _ = w.Write([]byte(postSuccessTexts[r.URL.Path]))
		})
	}

	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	ctx := context.Background()
	if err := client.KickPlayer(ctx, "p1", ""); err != nil {
		t.Fatalf("KickPlayer failed: %v", err)
	}
	if err := client.BanPlayer(ctx, "p2", ""); err != nil {
		t.Fatalf("BanPlayer failed: %v", err)
	}
	if err := client.ShutdownServer(ctx, 5, ""); err != nil {
		t.Fatalf("ShutdownServer failed: %v", err)
	}

	want := map[string]string{
		"/v1/api/kick":     `{"userid":"p1"}`,
		"/v1/api/ban":      `{"userid":"p2"}`,
		"/v1/api/shutdown": `{"waittime":5}`,
	}
	for path, expected := range want {
		mu.Lock()
		got := payloads[path]
		mu.Unlock()
		if got != expected {
			t.Fatalf("payload on %s = %s, want %s", path, got, expected)
		}
	}
}

func TestClient_ShutdownServer_NegativeWaitTime(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if err := client.ShutdownServer(context.Background(), -1, "bye"); err == nil {
		t.Fatal("expected error for negative waittime")
	}
}

func TestClient_WithHTTPClientAndTimeout(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/info", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ServerInfo{ServerName: "Injected"})
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	httpClient := &http.Client{Timeout: 7 * time.Second}
	client, err := NewClient(srv.URL, "secret",
		WithHTTPClient(httpClient),
		WithTimeout(3*time.Second))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	info, err := client.GetServerInfo(context.Background())
	if err != nil {
		t.Fatalf("GetServerInfo failed: %v", err)
	}
	if info.ServerName != "Injected" {
		t.Fatalf("unexpected response: %+v", info)
	}
	if httpClient.Timeout != 7*time.Second {
		t.Fatalf("injected client timeout was mutated: %v", httpClient.Timeout)
	}
}

func TestClient_WithTimeoutOnInternalClient(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret", WithTimeout(42*time.Second))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}
	if client.client.Timeout != 42*time.Second {
		t.Fatalf("unexpected internal timeout: %v", client.client.Timeout)
	}
}

func TestClient_DefaultTimeout(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}
	if client.client.Timeout != defaultTimeout {
		t.Fatalf("unexpected default timeout: %v", client.client.Timeout)
	}
}

func TestClient_Close(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close should return nil: %v", err)
	}
}

func TestClient_Close_ClosesIdleConnections(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	tr := &trackingTransport{}
	client.client.Transport = tr
	if tr.closed {
		t.Fatal("idle connections closed before Close")
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close should return nil: %v", err)
	}
	if !tr.closed {
		t.Fatal("Close did not close idle connections")
	}
}

func TestClient_Close_DoesNotCloseInjectedClient(t *testing.T) {
	tr := &trackingTransport{}
	client, err := NewClient("127.0.0.1:17999", "secret", WithHTTPClient(&http.Client{Transport: tr}))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if err := client.Close(); err != nil {
		t.Fatalf("Close should return nil: %v", err)
	}
	if tr.closed {
		t.Fatal("Close closed idle connections of the injected client")
	}
}

func TestClient_WithHTTPClientNilFallsBackToInternal(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret", WithHTTPClient(nil))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}
	if client.client == nil {
		t.Fatal("expected an internal http.Client to be created")
	}
	if !client.ownClient {
		t.Fatal("expected the internal client to be owned by the Client")
	}
}

func TestClient_ResponseBodyLimit(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/info", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("a"), 200))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret", WithMaxResponseBytes(100))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	_, err = client.GetServerInfo(context.Background())
	if err == nil {
		t.Fatal("expected error for response over the configured limit")
	}
	if !strings.Contains(err.Error(), "response body exceeds 100 bytes") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_MaxIntResponseBytes(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/info", func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(ServerInfo{ServerName: "MaxInt"})
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret", WithMaxResponseBytes(math.MaxInt))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	info, err := client.GetServerInfo(context.Background())
	if err != nil {
		t.Fatalf("GetServerInfo failed: %v", err)
	}
	if info.ServerName != "MaxInt" {
		t.Fatalf("unexpected response: %+v", info)
	}
}

func TestClient_ErrorResponseReusesConnection(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write(bytes.Repeat([]byte("x"), 2048))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret")
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	var connects atomic.Int32
	ctx := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
		ConnectStart: func(_, _ string) { connects.Add(1) },
	})

	for i := 0; i < 3; i++ {
		if _, err := client.GetServerMetrics(ctx); err == nil {
			t.Fatal("expected APIError")
		}
	}
	if connects.Load() != 1 {
		t.Fatalf("drained error bodies should reuse the connection, got %d connections", connects.Load())
	}
}

func TestClient_OversizedResponseReusesConnection(t *testing.T) {
	handler := http.NewServeMux()
	handler.HandleFunc("/v1/api/info", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(bytes.Repeat([]byte("a"), 200))
	})
	srv := httptest.NewServer(handler)
	defer srv.Close()

	client, err := NewClient(srv.URL, "secret", WithMaxResponseBytes(100))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	var connects atomic.Int32
	ctx := httptrace.WithClientTrace(context.Background(), &httptrace.ClientTrace{
		ConnectStart: func(_, _ string) { connects.Add(1) },
	})

	for i := 0; i < 3; i++ {
		if _, err := client.GetServerInfo(ctx); err == nil {
			t.Fatal("expected oversized response error")
		}
	}
	if connects.Load() != 1 {
		t.Fatalf("drained oversized bodies should reuse the connection, got %d connections", connects.Load())
	}
}

type trackingTransport struct {
	closed bool
}

func (t *trackingTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("unused")
}

func (t *trackingTransport) CloseIdleConnections() {
	t.closed = true
}

// faultyReadCloser is an io.ReadCloser whose Read always returns the
// configured error, used to exercise the response-read error branches.
type faultyReadCloser struct {
	err error
}

func (f faultyReadCloser) Read([]byte) (int, error) {
	return 0, f.err
}

func (f faultyReadCloser) Close() error {
	return nil
}

// faultInjectionTransport returns a canned response for every request,
// optionally with a body that fails on Read or without a Content-Type.
type faultInjectionTransport struct {
	statusCode int
	body       io.ReadCloser
	readErr    error
	header     http.Header
}

func (t faultInjectionTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	body := t.body
	if body == nil {
		body = faultyReadCloser{err: t.readErr}
	}
	header := t.header
	if header == nil {
		header = make(http.Header)
	}
	return &http.Response{StatusCode: t.statusCode, Body: body, Header: header, Request: req}, nil
}

func TestClient_ReadErrorOnErrorResponse(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret", WithHTTPClient(&http.Client{
		Transport: faultInjectionTransport{statusCode: http.StatusInternalServerError, readErr: errors.New("read failed")},
	}))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	_, err = client.GetServerMetrics(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to read error response body") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_ReadErrorOnSuccessResponse(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret", WithHTTPClient(&http.Client{
		Transport: faultInjectionTransport{statusCode: http.StatusOK, readErr: errors.New("read failed")},
	}))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	_, err = client.GetServerInfo(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "failed to read response body") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_DecodeFailureWithoutContentType(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret", WithHTTPClient(&http.Client{
		Transport: faultInjectionTransport{
			statusCode: http.StatusOK,
			body:       io.NopCloser(strings.NewReader("not-json")),
		},
	}))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	_, err = client.GetServerInfo(context.Background())
	if err == nil {
		t.Fatal("expected decode error")
	}
	if !strings.Contains(err.Error(), `content-type "unknown"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestClient_APIError_ErrorString(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret", WithHTTPClient(&http.Client{
		Transport: faultInjectionTransport{
			statusCode: http.StatusBadRequest,
			body:       io.NopCloser(strings.NewReader(`{"error":"bad"}`)),
		},
	}))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	_, err = client.GetServerMetrics(context.Background())
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got: %v", err)
	}
	msg := apiErr.Error()
	for _, want := range []string{"rest api error:", "status=400", "method=GET", "path=/metrics"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("error string missing %q: %s", want, msg)
		}
	}
}

func TestClient_GetServerSettings_ErrorPath(t *testing.T) {
	client, err := NewClient("127.0.0.1:17999", "secret", WithHTTPClient(&http.Client{
		Transport: faultInjectionTransport{
			statusCode: http.StatusInternalServerError,
			body:       io.NopCloser(strings.NewReader(`{"error":"boom"}`)),
		},
	}))
	if err != nil {
		t.Fatalf("error creating client: %v", err)
	}

	if _, err := client.GetServerSettings(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewClient_ParseError(t *testing.T) {
	if _, err := NewClient("http://ho st:8212", "secret"); err == nil {
		t.Fatal("expected error for a base URL that fails to parse")
	}
}

func TestNewClient_MissingHost(t *testing.T) {
	if _, err := NewClient("http://", "secret"); err == nil {
		t.Fatal("expected error for a base URL without a host")
	}
}
