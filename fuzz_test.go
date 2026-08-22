package palrest

import (
	"bytes"
	"encoding/json"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func FuzzNormalizeBaseURL(f *testing.F) {
	for _, seed := range []string{
		"127.0.0.1:8212",
		"https://example.com:443",
		"[::1]:2222",
		"::1:2222",
		"fe80::1%eth0:2222",
		"http://example.com/path",
		"http://user:password@example.com:8212",
		"",
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		normalized, err := normalizeBaseURL(raw, defaultPort)
		if err != nil {
			return
		}
		assertNormalizedBaseURL(t, normalized)
	})
}

func assertNormalizedBaseURL(t *testing.T, normalized string) {
	t.Helper()

	u, err := url.Parse(normalized)
	if err != nil {
		t.Fatalf("accepted URL does not parse: %v", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		t.Fatalf("accepted URL has unsupported scheme: %q", u.Scheme)
	}
	if u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		t.Fatalf("accepted URL contains rejected components: %q", normalized)
	}
	if u.Hostname() == "" || !validHostname(u.Hostname()) {
		t.Fatalf("accepted URL has invalid hostname: %q", normalized)
	}
	if _, err := validatePort(u.Port()); err != nil {
		t.Fatalf("accepted URL has invalid port: %v", err)
	}
}

func FuzzActorUnmarshalJSON(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(`null`),
		[]byte(`{"Type":"Character","UnitType":"Player","LocationX":1}`),
		[]byte(`{"Type":"PalBox","LocationX":1}`),
		[]byte(`{"Type":"FutureKind","FutureField":true}`),
		[]byte(`{}`),
		[]byte(`[]`),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		assertFuzzActorState(t, data)
	})
}

func assertFuzzActorState(t *testing.T, data []byte) {
	t.Helper()

	var actor Actor
	if err := actor.UnmarshalJSON(data); err != nil {
		return
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		if actor != (Actor{}) {
			t.Fatalf("null did not leave actor zeroed: %+v", actor)
		}
		return
	}
	if actor.Type == "" {
		t.Fatal("successful non-null decode has empty Type")
	}
	assertFuzzActorPayload(t, actor)
}

func assertFuzzActorPayload(t *testing.T, actor Actor) {
	t.Helper()

	switch actor.Type {
	case "Character":
		if actor.Character == nil || actor.PalBox != nil {
			t.Fatalf("invalid Character actor state: %+v", actor)
		}
	case "PalBox":
		if actor.PalBox == nil || actor.Character != nil {
			t.Fatalf("invalid PalBox actor state: %+v", actor)
		}
	default:
		if actor.Character != nil || actor.PalBox != nil {
			t.Fatalf("unknown actor allocated a concrete payload: %+v", actor)
		}
	}
}

func FuzzDecodeResponseBody(f *testing.F) {
	for _, seed := range [][]byte{
		[]byte(``),
		[]byte(`null`),
		[]byte(`{"error":"bad"}`),
		[]byte(`not-json`),
		[]byte(" \r\n error \t "),
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		got := decodeResponseBody(data)
		if len(data) == 0 {
			if got != nil {
				t.Fatalf("empty body decoded as %#v", got)
			}
			return
		}

		var expected any
		if err := json.Unmarshal(data, &expected); err == nil {
			if !reflect.DeepEqual(got, expected) {
				t.Fatalf("JSON body decoded differently: got %#v, want %#v", got, expected)
			}
			return
		}
		if text, ok := got.(string); !ok || text != strings.TrimSpace(string(data)) {
			t.Fatalf("invalid JSON body was not preserved as trimmed text: %#v", got)
		}
	})
}
