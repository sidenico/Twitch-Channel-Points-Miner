package gql

import "testing"

func TestGetUserAgentReturnsNonEmpty(t *testing.T) {
	ua := GetUserAgent("ignored")
	if ua == "" {
		t.Fatalf("expected non-empty user agent")
	}
}
