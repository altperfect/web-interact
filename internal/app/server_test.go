package app

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCaptureSlug(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
		ok   bool
	}{
		{name: "webhook root", path: "/at/lunar-meteor-a7", want: "lunar-meteor-a7", ok: true},
		{name: "webhook child path", path: "/at/lunar-meteor-a7/orders/123", want: "lunar-meteor-a7", ok: true},
		{name: "wrong prefix", path: "/api/webhooks", ok: false},
		{name: "invalid slug", path: "/at/Lunar", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := captureSlug(tt.path)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("slug = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidatePublicID(t *testing.T) {
	if !validatePublicID("0123456789ABCDEFGHIJKLMNOPQRSTUV") {
		t.Fatal("expected valid base62 id")
	}
	if validatePublicID("short") {
		t.Fatal("expected short id to be invalid")
	}
	if validatePublicID("0123456789ABCDEFGHIJKLMNOPQRSTU!") {
		t.Fatal("expected non-base62 id to be invalid")
	}
}

func TestNormalizeWebhookDisplayName(t *testing.T) {
	got, err := normalizeWebhookDisplayName("  test  ")
	if err != nil {
		t.Fatalf("normalizeWebhookDisplayName() error = %v", err)
	}
	if got != "test" {
		t.Fatalf("normalizeWebhookDisplayName() = %q, want test", got)
	}
	if _, err := normalizeWebhookDisplayName(""); err == nil {
		t.Fatal("expected empty webhook name to be rejected")
	}
	if _, err := normalizeWebhookDisplayName(strings.Repeat("x", maxWebhookNameRunes+1)); err == nil {
		t.Fatal("expected long webhook name to be rejected")
	}
	if _, err := normalizeWebhookDisplayName("bad\nname"); err == nil {
		t.Fatal("expected control character in webhook name to be rejected")
	}
}

func TestNormalizeRequestNote(t *testing.T) {
	got, err := normalizeRequestNote("  remember this\nline  ")
	if err != nil {
		t.Fatalf("normalizeRequestNote() error = %v", err)
	}
	if got != "remember this\nline" {
		t.Fatalf("normalizeRequestNote() = %q, want trimmed multiline note", got)
	}
	if _, err := normalizeRequestNote(strings.Repeat("x", maxRequestNoteRunes+1)); err == nil {
		t.Fatal("expected long request note to be rejected")
	}
	if _, err := normalizeRequestNote("bad\x00note"); err == nil {
		t.Fatal("expected NUL in request note to be rejected")
	}
}

func TestShareTokenChangesWithSecret(t *testing.T) {
	webhook := Webhook{Slug: "silent-comet-a7", ShareNonce: "nonce"}
	a := New(Config{AppSecret: []byte("0123456789abcdef0123456789abcdef")}, nil)
	b := New(Config{AppSecret: []byte("abcdef0123456789abcdef0123456789")}, nil)
	if a.shareToken(webhook) == b.shareToken(webhook) {
		t.Fatal("share token should depend on APP_SECRET")
	}
}

func TestNormalizeResponseContentType(t *testing.T) {
	if got := normalizeResponseContentType(""); got != defaultWebhookResponseType {
		t.Fatalf("empty content type = %q, want %q", got, defaultWebhookResponseType)
	}
	if got := normalizeResponseContentType(" text/html "); got != "text/html" {
		t.Fatalf("trimmed content type = %q, want text/html", got)
	}
	if validHeaderValue("text/plain\r\nX-Test: bad") {
		t.Fatal("expected CRLF content type to be invalid")
	}
}

func TestResponseStatusAndRedirectValidation(t *testing.T) {
	if validResponseStatusCode(999) {
		t.Fatal("expected 999 to be rejected")
	}
	if !validResponseStatusCode(302) {
		t.Fatal("expected 302 to be valid")
	}

	app := New(Config{}, nil)
	req := httptest.NewRequest("PATCH", "http://example.test/api/webhooks/silent-comet-a7/response", nil)
	if _, err := app.normalizeResponseLocation(req, "silent-comet-a7", 302, "/at/silent-comet-a7"); err == nil {
		t.Fatal("expected redirect to same webhook to be rejected")
	}
	if got, err := app.normalizeResponseLocation(req, "silent-comet-a7", 302, "/thanks"); err != nil || got != "/thanks" {
		t.Fatalf("safe relative redirect = %q, %v; want /thanks, nil", got, err)
	}
	if got, err := app.normalizeResponseLocation(req, "silent-comet-a7", 200, "/at/silent-comet-a7"); err != nil || got != "" {
		t.Fatalf("non-redirect location = %q, %v; want empty, nil", got, err)
	}
}

func TestNormalizeExtraResponseHeaders(t *testing.T) {
	headers, err := normalizeExtraResponseHeaders([]ResponseHeader{{Name: "x-test", Value: "ok"}})
	if err != nil {
		t.Fatalf("normalizeExtraResponseHeaders() error = %v", err)
	}
	if headers[0].Name != "X-Test" || headers[0].Value != "ok" {
		t.Fatalf("headers[0] = %#v, want canonical X-Test", headers[0])
	}
	if _, err := normalizeExtraResponseHeaders([]ResponseHeader{{Name: "Location", Value: "/next"}}); err == nil {
		t.Fatal("expected reserved Location header to be rejected")
	}
}

func TestShouldReuseTelegramProxyPassword(t *testing.T) {
	existing := TelegramSettings{
		ProxyEnabled:  true,
		ProxyHost:     "127.0.0.1",
		ProxyPort:     9050,
		ProxyUsername: "proxy-user",
		ProxyPassword: "saved-secret",
	}

	tests := []struct {
		name        string
		settings    TelegramSettings
		hasExisting bool
		want        bool
	}{
		{
			name: "same saved proxy",
			settings: TelegramSettings{
				ProxyHost:     "127.0.0.1",
				ProxyPort:     9050,
				ProxyUsername: "proxy-user",
			},
			hasExisting: true,
			want:        true,
		},
		{
			name: "changed username",
			settings: TelegramSettings{
				ProxyHost:     "127.0.0.1",
				ProxyPort:     9050,
				ProxyUsername: "other-user",
			},
			hasExisting: true,
			want:        false,
		},
		{
			name: "changed host",
			settings: TelegramSettings{
				ProxyHost:     "10.0.0.1",
				ProxyPort:     9050,
				ProxyUsername: "proxy-user",
			},
			hasExisting: true,
			want:        false,
		},
		{
			name: "no saved settings",
			settings: TelegramSettings{
				ProxyHost:     "127.0.0.1",
				ProxyPort:     9050,
				ProxyUsername: "proxy-user",
			},
			hasExisting: false,
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldReuseTelegramProxyPassword(tt.settings, existing, tt.hasExisting); got != tt.want {
				t.Fatalf("shouldReuseTelegramProxyPassword() = %v, want %v", got, tt.want)
			}
		})
	}
}
