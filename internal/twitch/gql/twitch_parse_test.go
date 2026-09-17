package gql

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"TwitchChannelPointsMiner/internal/constants"
	"TwitchChannelPointsMiner/internal/streamer"
	"TwitchChannelPointsMiner/internal/twitch/auth"
)

func newFlexibleTwitch(t *testing.T, responder func(operation string, body string) (*http.Response, error)) *Twitch {
	t.Helper()
	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			bodyBytes, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("read request body: %v", err)
			}
			_ = req.Body.Close()
			body := string(bodyBytes)
			operation := ""
			var envelope struct {
				OperationName string `json:"operationName"`
			}
			if err := json.Unmarshal(bodyBytes, &envelope); err == nil {
				operation = envelope.OperationName
			}
			return responder(operation, body)
		}),
	}
	return &Twitch{
		userAgent:      "ua",
		deviceID:       "device",
		clientSession:  "session",
		clientVersion:  constants.ClientVersion,
		versionTTL:     time.Hour,
		versionFetched: time.Now(),
		twitchLogin:    auth.NewTwitchLoginWithIdentity("token", "user-id"),
		client:         client,
	}
}

func TestNavigateMissingIntermediateKeys(t *testing.T) {
	data := map[string]interface{}{
		"data": map[string]interface{}{
			"user": map[string]interface{}{
				"id": "42",
			},
		},
	}
	if got := navigate(data, "data.user.id"); got != "42" {
		t.Fatalf("navigate id got %#v", got)
	}
	if got := navigate(data, "data.missing.id"); got != nil {
		t.Fatalf("missing intermediate should be nil, got %#v", got)
	}
	if got := navigate(nil, "data.user.id"); got != nil {
		t.Fatalf("nil root should be nil")
	}
	if got := navigate("not-a-map", "data"); got != nil {
		t.Fatalf("non-map root should be nil")
	}
}

func TestPostGQLDecodeMalformedBody(t *testing.T) {
	tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
		return jsonResponse(200, `{not-json`), nil
	})
	var out map[string]interface{}
	err := tw.PostGQLDecode(map[string]interface{}{"operationName": "X"}, &out)
	if err == nil {
		t.Fatalf("expected decode error for malformed body")
	}
}

func TestGetChannelIDMissingUser(t *testing.T) {
	tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
		if operation != "CoreActionsChannelLogin" && !strings.Contains(body, "GetIDFromLogin") && operation != constants.GQLOperations.GetIDFromLogin.OperationName {
			// Accept whatever operation name GetIDFromLogin uses
		}
		return jsonResponse(200, `{"data":{}}`), nil
	})
	_, err := tw.GetChannelID("someone")
	if err == nil {
		t.Fatalf("expected not-found error")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetChannelIDParsesUserID(t *testing.T) {
	tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
		return jsonResponse(200, `{"data":{"user":{"id":"998877"}}}`), nil
	})
	id, err := tw.GetChannelID("someone")
	if err != nil {
		t.Fatalf("GetChannelID: %v", err)
	}
	if id != "998877" {
		t.Fatalf("id got %q", id)
	}
}

func TestGetFollowersEmptyTreeReturnsNilSlice(t *testing.T) {
	tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
		return jsonResponse(200, `{"data":{"user":{}}}`), nil
	})
	follows, err := tw.GetFollowers(10, streamer.FollowersOrderDESC)
	if err != nil {
		t.Fatalf("GetFollowers: %v", err)
	}
	if len(follows) != 0 {
		t.Fatalf("expected empty follows, got %#v", follows)
	}
}

func TestGetFollowersParsesEdges(t *testing.T) {
	tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
		return jsonResponse(200, `{
			"data": {
				"user": {
					"follows": {
						"edges": [
							{"node": {"login": "Alice"}, "cursor": "c1"},
							{"node": {"login": "Bob"}, "cursor": "c2"}
						],
						"pageInfo": {"hasNextPage": false}
					}
				}
			}
		}`), nil
	})
	follows, err := tw.GetFollowers(10, streamer.FollowersOrderDESC)
	if err != nil {
		t.Fatalf("GetFollowers: %v", err)
	}
	if len(follows) != 2 || follows[0] != "alice" || follows[1] != "bob" {
		t.Fatalf("follows got %#v", follows)
	}
}

func TestStreamInfoOverlayNullUserVsNullStream(t *testing.T) {
	t.Run("null user", func(t *testing.T) {
		tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
			return jsonResponse(200, `{"data":{"user":null}}`), nil
		})
		_, err := tw.streamInfoOverlay("missing", "1")
		if err == nil || !errors.Is(err, ErrChannelNotFound) {
			t.Fatalf("expected ErrChannelNotFound, got %v", err)
		}
	})
	t.Run("null stream", func(t *testing.T) {
		tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
			return jsonResponse(200, `{
				"data": {
					"user": {
						"stream": null,
						"broadcastSettings": null
					}
				}
			}`), nil
		})
		_, err := tw.streamInfoOverlay("offline", "1")
		if err == nil || !errors.Is(err, ErrStreamerOffline) {
			t.Fatalf("expected ErrStreamerOffline, got %v", err)
		}
	})
	t.Run("live stream", func(t *testing.T) {
		tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
			return jsonResponse(200, `{
				"data": {
					"user": {
						"stream": {
							"id": "stream-1",
							"viewersCount": 12,
							"createdAt": "2026-03-01T10:00:00Z",
							"tags": [{"id": "tag-1"}]
						},
						"broadcastSettings": {
							"title": "hello",
							"game": {"id": "g1", "name": "Game", "displayName": "Game"}
						}
					}
				}
			}`), nil
		})
		info, err := tw.streamInfoOverlay("live", "1")
		if err != nil {
			t.Fatalf("streamInfoOverlay: %v", err)
		}
		if info.StreamID != "stream-1" || info.Title != "hello" || info.ViewersCount != 12 {
			t.Fatalf("unexpected info %#v", info)
		}
	})
}

func TestIsStreamLiveMalformedAndStates(t *testing.T) {
	t.Run("missing channel id", func(t *testing.T) {
		tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
			t.Fatalf("should not call network")
			return nil, nil
		})
		_, err := tw.IsStreamLive("")
		if err == nil {
			t.Fatalf("expected error")
		}
	})
	t.Run("null user means not live", func(t *testing.T) {
		tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
			return jsonResponse(200, `{"data":{"user":null}}`), nil
		})
		live, err := tw.IsStreamLive("1")
		if err != nil {
			t.Fatalf("IsStreamLive: %v", err)
		}
		if live {
			t.Fatalf("expected not live")
		}
	})
	t.Run("live", func(t *testing.T) {
		tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
			return jsonResponse(200, `{"data":{"user":{"stream":{"id":"s","createdAt":"2026-03-01T10:00:00Z"}}}}`), nil
		})
		live, err := tw.IsStreamLive("1")
		if err != nil {
			t.Fatalf("IsStreamLive: %v", err)
		}
		if !live {
			t.Fatalf("expected live")
		}
	})
	t.Run("malformed body", func(t *testing.T) {
		tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
			return jsonResponse(200, `{`), nil
		})
		_, err := tw.IsStreamLive("1")
		if err == nil {
			t.Fatalf("expected decode error")
		}
	})
}

func TestLoadChannelPointsContextMalformedAndMissingChannel(t *testing.T) {
	t.Run("malformed", func(t *testing.T) {
		tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
			return jsonResponse(200, `{`), nil
		})
		s := &streamer.Streamer{Username: "x", Stream: streamer.NewStream()}
		_, err := tw.LoadChannelPointsContext(s)
		if err == nil {
			t.Fatalf("expected error")
		}
	})
	t.Run("missing channel", func(t *testing.T) {
		tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
			return jsonResponse(200, `{"data":{"community":{"channel":null}}}`), nil
		})
		s := &streamer.Streamer{Username: "x", Stream: streamer.NewStream()}
		_, err := tw.LoadChannelPointsContext(s)
		if err == nil || !errors.Is(err, ErrChannelNotFound) {
			t.Fatalf("expected ErrChannelNotFound, got %v", err)
		}
	})
	t.Run("parses balance and multipliers", func(t *testing.T) {
		tw := newFlexibleTwitch(t, func(operation string, body string) (*http.Response, error) {
			return jsonResponse(200, `{
				"data": {
					"community": {
						"channel": {
							"self": {
								"communityPoints": {
									"balance": 1234,
									"activeMultipliers": [{"factor": 1.2}],
									"availableClaim": null
								}
							},
							"communityPointsSettings": {"goals": []}
						}
					}
				}
			}`), nil
		})
		s := &streamer.Streamer{Username: "x", Stream: streamer.NewStream()}
		balance, err := tw.LoadChannelPointsContext(s)
		if err != nil {
			t.Fatalf("LoadChannelPointsContext: %v", err)
		}
		if balance != 1234 || s.ChannelPoints != 1234 {
			t.Fatalf("balance got %d streamer=%d", balance, s.ChannelPoints)
		}
		if !s.HasActiveMultipliers() {
			t.Fatalf("expected active multipliers")
		}
	})
}

func TestParseCommunityGoalsMalformed(t *testing.T) {
	if got := parseCommunityGoals(nil); got != nil {
		t.Fatalf("nil goals got %#v", got)
	}
	if got := parseCommunityGoals("nope"); got != nil {
		t.Fatalf("non-array goals got %#v", got)
	}
	goals := parseCommunityGoals([]interface{}{
		map[string]interface{}{"id": "g1", "title": "Goal", "pointsContributed": 1.0, "amountNeeded": 10.0},
		"skip-me",
		map[string]interface{}{"title": "no-id"},
	})
	if len(goals) != 1 || goals["g1"] == nil {
		t.Fatalf("expected one valid goal, got %#v", goals)
	}
}

func TestParseRFC3339TimestampEmptyAndInvalid(t *testing.T) {
	if !parseRFC3339Timestamp("").IsZero() {
		t.Fatalf("empty should be zero")
	}
	if !parseRFC3339Timestamp("not-a-time").IsZero() {
		t.Fatalf("invalid should be zero")
	}
	got := parseRFC3339Timestamp("2026-03-01T10:00:00Z")
	if got.IsZero() {
		t.Fatalf("valid timestamp should parse")
	}
}
