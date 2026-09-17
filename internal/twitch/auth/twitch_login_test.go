package auth

import (
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeCookieStoreModernFormat(t *testing.T) {
	raw := []byte(`{
		"auth-token": {"value": "oauth-token"},
		"persistent": {"value": "12345", "path": "/", "domain": ".twitch.tv"},
		"other": {"value": "abc"}
	}`)
	store, err := decodeCookieStore(raw)
	if err != nil {
		t.Fatalf("decodeCookieStore: %v", err)
	}
	if store["auth-token"].Value != "oauth-token" {
		t.Fatalf("auth-token got %#v", store["auth-token"])
	}
	if store["persistent"].Value != "12345" {
		t.Fatalf("persistent got %#v", store["persistent"])
	}
	if store["other"].Value != "abc" {
		t.Fatalf("other got %#v", store["other"])
	}
}

func TestDecodeCookieStoreLegacyArrayFormat(t *testing.T) {
	raw := []byte(`[
		{"name":"auth-token","value":"legacy-token","path":"/","domain":".twitch.tv"},
		{"name":"persistent","value":"999","path":"/","domain":".twitch.tv"},
		{"name":"","value":"skip-me"}
	]`)
	store, err := decodeCookieStore(raw)
	if err != nil {
		t.Fatalf("decodeCookieStore: %v", err)
	}
	if store["auth-token"].Value != "legacy-token" {
		t.Fatalf("auth-token got %#v", store["auth-token"])
	}
	if store["persistent"].Value != "999" {
		t.Fatalf("persistent got %#v", store["persistent"])
	}
	if _, ok := store[""]; ok {
		t.Fatalf("empty name should be skipped")
	}
}

func TestDecodeCookieStoreRejectsGarbage(t *testing.T) {
	if _, err := decodeCookieStore([]byte(`not-json`)); err == nil {
		t.Fatalf("expected error for garbage input")
	}
}

func TestLoadAndSaveCookiesRoundTripWithoutNetwork(t *testing.T) {
	login, err := NewTwitchLogin("client", "device", "tester", "ua", "")
	if err != nil {
		t.Fatalf("NewTwitchLogin: %v", err)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "cookies", "tester.json")
	fixture := map[string]persistedCookie{
		"auth-token": {Value: "saved-oauth"},
		"persistent": {Value: "user-42", Path: "/", Domain: ".twitch.tv"},
		"session":    {Value: "sess", Path: "/", Domain: ".twitch.tv"},
	}
	raw, err := json.MarshalIndent(fixture, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	if err := login.loadCookies(path); err != nil {
		t.Fatalf("loadCookies: %v", err)
	}
	if login.AuthToken() != "saved-oauth" {
		t.Fatalf("AuthToken got %q", login.AuthToken())
	}
	if login.UserID() != "user-42" {
		t.Fatalf("UserID got %q want user-42", login.UserID())
	}

	outPath := filepath.Join(dir, "cookies", "tester-out.json")
	if err := login.saveCookies(outPath); err != nil {
		t.Fatalf("saveCookies: %v", err)
	}
	saved, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read saved: %v", err)
	}
	var store cookieStore
	if err := json.Unmarshal(saved, &store); err != nil {
		t.Fatalf("saved cookies not modern map JSON: %v", err)
	}
	if store["auth-token"].Value != "saved-oauth" {
		t.Fatalf("saved auth-token got %#v", store["auth-token"])
	}
	if store["persistent"].Value != "user-42" {
		t.Fatalf("saved persistent got %#v", store["persistent"])
	}

	u, err := url.Parse("https://twitch.tv")
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	foundSession := false
	for _, c := range login.Client().Jar.Cookies(u) {
		if c.Name == "session" && c.Value == "sess" {
			foundSession = true
		}
	}
	if !foundSession {
		names := make([]string, 0)
		for _, c := range login.Client().Jar.Cookies(u) {
			names = append(names, c.Name+"="+c.Value)
		}
		t.Fatalf("expected session cookie in jar, got %#v", names)
	}
}

func TestLoadCookiesMissingFile(t *testing.T) {
	login, err := NewTwitchLogin("client", "device", "tester", "ua", "")
	if err != nil {
		t.Fatalf("NewTwitchLogin: %v", err)
	}
	err = login.loadCookies(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatalf("expected error for missing cookie file")
	}
	if !os.IsNotExist(err) {
		if _, ok := err.(*os.PathError); !ok {
			t.Fatalf("expected not-exist style error, got %T %v", err, err)
		}
	}
}
