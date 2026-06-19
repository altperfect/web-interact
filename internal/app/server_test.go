package app

import "testing"

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

func TestShareTokenChangesWithSecret(t *testing.T) {
	webhook := Webhook{Slug: "silent-comet-a7", ShareNonce: "nonce"}
	a := New(Config{AppSecret: []byte("0123456789abcdef0123456789abcdef")}, nil)
	b := New(Config{AppSecret: []byte("abcdef0123456789abcdef0123456789")}, nil)
	if a.shareToken(webhook) == b.shareToken(webhook) {
		t.Fatal("share token should depend on APP_SECRET")
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
