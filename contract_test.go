package palrest

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const (
	contractVersion  = "v1.0.3"
	contractPrefix   = "/v1/api"
	contractUser     = "admin"
	contractPassword = "contract-password"
)

type contractEndpoint struct {
	name                string
	method              string
	path                string
	requestBody         string
	responseBody        string
	responseContentType string
	errorStatuses       []int
	invoke              func(*Client) error
}

func TestDocumentedRESTContract_Success(t *testing.T) {
	for _, endpoint := range documentedContractEndpoints(t) {
		t.Run(endpoint.name, func(t *testing.T) {
			server := newContractServer(t, endpoint, http.StatusOK)
			defer server.Close()

			client, err := NewClient(server.URL, contractPassword)
			if err != nil {
				t.Fatalf("NewClient failed: %v", err)
			}
			if err := endpoint.invoke(client); err != nil {
				t.Fatalf("endpoint failed: %v", err)
			}
		})
	}
}

func TestDocumentedRESTContract_ErrorStatuses(t *testing.T) {
	for _, endpoint := range documentedContractEndpoints(t) {
		for _, status := range endpoint.errorStatuses {
			status := status
			t.Run(fmt.Sprintf("%s/%d", endpoint.name, status), func(t *testing.T) {
				assertContractError(t, endpoint, status)
			})
		}
	}
}

func assertContractError(t *testing.T, endpoint contractEndpoint, status int) {
	t.Helper()

	server := newContractServer(t, endpoint, status)
	defer server.Close()

	client, err := NewClient(server.URL, contractPassword)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}
	err = endpoint.invoke(client)
	if err == nil {
		t.Fatalf("expected status %d error", status)
	}

	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected APIError, got %T: %v", err, err)
	}
	if apiErr.StatusCode != status {
		t.Fatalf("unexpected status: got %d, want %d", apiErr.StatusCode, status)
	}
}

func documentedContractEndpoints(t *testing.T) []contractEndpoint {
	t.Helper()

	postResponses := readContractResponses(t)
	return []contractEndpoint{
		{
			name:                "info",
			method:              http.MethodGet,
			path:                "/info",
			responseBody:        readContractFixture(t, "info.json"),
			responseContentType: "application/json",
			errorStatuses:       []int{http.StatusBadRequest, http.StatusUnauthorized},
			invoke: func(client *Client) error {
				_, err := client.GetServerInfo(context.Background())
				return err
			},
		},
		{
			name:                "players",
			method:              http.MethodGet,
			path:                "/players",
			responseBody:        readContractFixture(t, "players.json"),
			responseContentType: "application/json",
			errorStatuses:       []int{http.StatusBadRequest, http.StatusUnauthorized},
			invoke: func(client *Client) error {
				_, err := client.GetPlayerList(context.Background())
				return err
			},
		},
		{
			name:                "settings",
			method:              http.MethodGet,
			path:                "/settings",
			responseBody:        readContractFixture(t, "settings.json"),
			responseContentType: "application/json",
			errorStatuses:       []int{http.StatusBadRequest, http.StatusUnauthorized},
			invoke: func(client *Client) error {
				_, err := client.GetServerSettings(context.Background())
				return err
			},
		},
		{
			name:                "metrics",
			method:              http.MethodGet,
			path:                "/metrics",
			responseBody:        readContractFixture(t, "metrics.json"),
			responseContentType: "application/json",
			errorStatuses:       []int{http.StatusBadRequest, http.StatusUnauthorized},
			invoke: func(client *Client) error {
				_, err := client.GetServerMetrics(context.Background())
				return err
			},
		},
		{
			name:                "game-data",
			method:              http.MethodGet,
			path:                "/game-data",
			responseBody:        readContractFixture(t, "game-data.json"),
			responseContentType: "application/json",
			errorStatuses:       []int{http.StatusUnauthorized},
			invoke: func(client *Client) error {
				_, err := client.GetGameData(context.Background())
				return err
			},
		},
		{
			name:          "announce",
			method:        http.MethodPost,
			path:          "/announce",
			requestBody:   `{"message":"Server maintenance"}`,
			responseBody:  postResponses["/announce"],
			errorStatuses: []int{http.StatusBadRequest, http.StatusUnauthorized},
			invoke: func(client *Client) error {
				return client.MakeAnnouncement(context.Background(), "Server maintenance")
			},
		},
		{
			name:          "kick",
			method:        http.MethodPost,
			path:          "/kick",
			requestBody:   `{"userid":"steam_123","message":"Please reconnect"}`,
			responseBody:  postResponses["/kick"],
			errorStatuses: []int{http.StatusBadRequest, http.StatusUnauthorized},
			invoke: func(client *Client) error {
				return client.KickPlayer(context.Background(), "steam_123", "Please reconnect")
			},
		},
		{
			name:          "ban",
			method:        http.MethodPost,
			path:          "/ban",
			requestBody:   `{"userid":"steam_123","message":"Rule violation"}`,
			responseBody:  postResponses["/ban"],
			errorStatuses: []int{http.StatusBadRequest, http.StatusUnauthorized},
			invoke: func(client *Client) error {
				return client.BanPlayer(context.Background(), "steam_123", "Rule violation")
			},
		},
		{
			name:          "unban",
			method:        http.MethodPost,
			path:          "/unban",
			requestBody:   `{"userid":"steam_123"}`,
			responseBody:  postResponses["/unban"],
			errorStatuses: []int{http.StatusBadRequest, http.StatusUnauthorized},
			invoke: func(client *Client) error {
				return client.UnbanPlayer(context.Background(), "steam_123")
			},
		},
		{
			name:          "save",
			method:        http.MethodPost,
			path:          "/save",
			responseBody:  postResponses["/save"],
			errorStatuses: []int{http.StatusBadRequest, http.StatusUnauthorized},
			invoke: func(client *Client) error {
				return client.SaveServerState(context.Background())
			},
		},
		{
			name:          "shutdown",
			method:        http.MethodPost,
			path:          "/shutdown",
			requestBody:   `{"waittime":30,"message":"Scheduled maintenance"}`,
			responseBody:  postResponses["/shutdown"],
			errorStatuses: []int{http.StatusBadRequest, http.StatusUnauthorized},
			invoke: func(client *Client) error {
				return client.ShutdownServer(context.Background(), 30, "Scheduled maintenance")
			},
		},
		{
			name:          "stop",
			method:        http.MethodPost,
			path:          "/stop",
			responseBody:  postResponses["/stop"],
			errorStatuses: []int{http.StatusBadRequest, http.StatusUnauthorized},
			invoke: func(client *Client) error {
				return client.StopServer(context.Background())
			},
		},
	}
}

func newContractServer(t *testing.T, endpoint contractEndpoint, status int) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assertContractRequestMetadata(t, endpoint, r)
		assertContractRequestBody(t, endpoint, r)
		writeContractResponse(w, endpoint, status)
	}))
}

func assertContractRequestMetadata(t *testing.T, endpoint contractEndpoint, r *http.Request) {
	t.Helper()

	if r.Method != endpoint.method {
		t.Errorf("%s: method = %s, want %s", endpoint.name, r.Method, endpoint.method)
	}
	if r.URL.Path != contractPrefix+endpoint.path {
		t.Errorf("%s: path = %s, want %s", endpoint.name, r.URL.Path, contractPrefix+endpoint.path)
	}
	username, password, ok := r.BasicAuth()
	if !ok || username != contractUser || password != contractPassword {
		t.Errorf("%s: unexpected basic auth: username=%q authenticated=%t", endpoint.name, username, ok)
	}
	if got := r.Header.Get("Accept"); got != "application/json" {
		t.Errorf("%s: Accept = %q, want application/json", endpoint.name, got)
	}
	if got := r.Header.Get("User-Agent"); got != "palrest-go" {
		t.Errorf("%s: User-Agent = %q, want palrest-go", endpoint.name, got)
	}
}

func assertContractRequestBody(t *testing.T, endpoint contractEndpoint, r *http.Request) {
	t.Helper()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		t.Errorf("%s: reading request body: %v", endpoint.name, err)
		return
	}
	if endpoint.requestBody == "" {
		if len(body) != 0 {
			t.Errorf("%s: unexpected request body: %s", endpoint.name, body)
		}
		if got := r.Header.Get("Content-Type"); got != "" {
			t.Errorf("%s: unexpected Content-Type for empty body: %q", endpoint.name, got)
		}
		return
	}
	if got := r.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("%s: Content-Type = %q, want application/json", endpoint.name, got)
	}
	assertContractJSON(t, endpoint.name+" request", body, []byte(endpoint.requestBody))
}

func writeContractResponse(w http.ResponseWriter, endpoint contractEndpoint, status int) {
	if status != http.StatusOK {
		w.WriteHeader(status)
		_, _ = io.WriteString(w, `{"error":"request failed"}`)
		return
	}
	if endpoint.responseContentType != "" {
		w.Header().Set("Content-Type", endpoint.responseContentType)
	}
	w.WriteHeader(http.StatusOK)
	_, _ = io.WriteString(w, endpoint.responseBody)
}

func assertContractJSON(t *testing.T, name string, got, want []byte) {
	t.Helper()

	var gotValue, wantValue any
	if err := json.Unmarshal(got, &gotValue); err != nil {
		t.Fatalf("%s: invalid JSON: %v", name, err)
	}
	if err := json.Unmarshal(want, &wantValue); err != nil {
		t.Fatalf("%s: invalid expected JSON: %v", name, err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Fatalf("%s: got %s, want %s", name, got, want)
	}
}

func readContractFixture(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join("testdata", "contract", contractVersion, name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read contract fixture %s: %v", path, err)
	}
	return strings.TrimSpace(string(data))
}

func readContractResponses(t *testing.T) map[string]string {
	t.Helper()

	var responses map[string]string
	data := readContractFixture(t, "post-success.json")
	if err := json.Unmarshal([]byte(data), &responses); err != nil {
		t.Fatalf("decode POST response fixtures: %v", err)
	}
	return responses
}
