package palrest_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/tagamer-net/palrest-go"
)

func ExampleClient_GetServerInfo() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{"version":"v0.1.5.0","servername":"Example Server","worldguid":"A7E97BAA"}`)
	}))
	defer srv.Close()

	client, err := palrest.NewClient(srv.URL, "admin-password")
	if err != nil {
		fmt.Println("client error:", err)
		return
	}
	defer func() { _ = client.Close() }()

	info, err := client.GetServerInfo(context.Background())
	if err != nil {
		fmt.Println("request error:", err)
		return
	}
	fmt.Printf("%s runs version %s", info.ServerName, info.Version)
	// Output: Example Server runs version v0.1.5.0
}

func ExampleClient_MakeAnnouncement() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		_, _ = fmt.Fprint(w, "The message was announced.")
	}))
	defer srv.Close()

	client, err := palrest.NewClient(srv.URL, "admin-password")
	if err != nil {
		fmt.Println("client error:", err)
		return
	}
	defer func() { _ = client.Close() }()

	err = client.MakeAnnouncement(context.Background(), "Server restart in 5 minutes")
	if err != nil {
		fmt.Println("request error:", err)
		return
	}
	fmt.Println("announcement sent")
	// Output: announcement sent
}

func ExampleClient_GetGameData() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprint(w, `{
			"Time": "2026-06-17 13:00:40",
			"FPS": 60,
			"AverageFPS": 30,
			"ActorData": [
				{"Type": "PalBox", "GuildID": "guild-1"},
				{"Type": "Character", "UnitType": "Player", "InstanceID": "char-1", "NickName": "PalUser"}
			]
		}`)
	}))
	defer srv.Close()

	client, err := palrest.NewClient(srv.URL, "admin-password",
		palrest.WithTimeout(60*time.Second),
		palrest.WithMaxResponseBytes(64<<20))
	if err != nil {
		fmt.Println("client error:", err)
		return
	}
	defer func() { _ = client.Close() }()

	data, err := client.GetGameData(context.Background())
	if err != nil {
		fmt.Println("request error:", err)
		return
	}
	for _, actor := range data.ActorData {
		if actor.Character == nil {
			continue
		}
		fmt.Printf("%s (%s) is active in the world", actor.Character.NickName, actor.Character.InstanceID)
		return
	}
	fmt.Println("no character actor found")
	// Output: PalUser (char-1) is active in the world
}

func ExampleClient_GetServerMetrics() {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = fmt.Fprint(w, `{"error":"invalid credentials"}`)
	}))
	defer srv.Close()

	client, err := palrest.NewClient(srv.URL, "wrong-password")
	if err != nil {
		fmt.Println("client error:", err)
		return
	}
	defer func() { _ = client.Close() }()

	_, err = client.GetServerMetrics(context.Background())
	var apiErr *palrest.APIError
	if errors.As(err, &apiErr) {
		fmt.Printf("API error: status %d on %s %s", apiErr.StatusCode, apiErr.Method, apiErr.Path)
	}
	// Output: API error: status 401 on GET /metrics
}
