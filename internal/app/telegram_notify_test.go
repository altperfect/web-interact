package app

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestBuildTelegramRequestMessagesIncludesDetailLink(t *testing.T) {
	headers, err := json.Marshal(map[string][]string{
		"User-Agent": {"curl/8.0"},
	})
	if err != nil {
		t.Fatal(err)
	}

	detailURL := "https://webhooks.altperfect.com/at/magnetic-meteor-ig/bkceK0Y1VdbM5BQOTla13G94p7FjwkJk"
	messages := buildTelegramRequestMessages(
		Webhook{Slug: "magnetic-meteor-ig"},
		CapturedRequest{
			PublicID:      "bkceK0Y1VdbM5BQOTla13G94p7FjwkJk",
			Method:        "POST",
			Path:          "/at/magnetic-meteor-ig/login",
			RemoteIP:      "203.0.113.10",
			Headers:       headers,
			Body:          []byte("hello"),
			CreatedAt:     time.Date(2026, 6, 19, 12, 34, 56, 0, time.UTC),
			ContentLength: 5,
		},
		"webhooks.altperfect.com",
		detailURL,
	)

	if len(messages) != 1 {
		t.Fatalf("message count = %d, want 1", len(messages))
	}
	if !strings.Contains(messages[0], "[View here]("+detailURL+")") {
		t.Fatalf("telegram message does not include detail link:\n%s", messages[0])
	}
	if !strings.Contains(messages[0], "POST /at/magnetic-meteor-ig/login HTTP/1.1") {
		t.Fatalf("telegram message does not include formatted request:\n%s", messages[0])
	}
}
