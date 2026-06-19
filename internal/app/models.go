package app

import (
	"encoding/json"
	"time"
)

type Owner struct {
	ID        int64
	TokenHash string
	CreatedAt time.Time
}

type Webhook struct {
	ID              int64
	Slug            string
	OwnerID         int64
	ShareEnabled    bool
	TelegramEnabled bool
	ShareNonce      string
	CreatedAt       time.Time
	UpdatedAt       time.Time
	RequestCount    int64
	LastRequestAt   *time.Time
}

type TelegramSettings struct {
	OwnerID       int64
	BotToken      string
	ChatID        string
	ProxyEnabled  bool
	ProxyHost     string
	ProxyPort     int
	ProxyUsername string
	ProxyPassword string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type CapturedRequest struct {
	ID            int64
	WebhookID     int64
	PublicID      string
	Method        string
	Path          string
	QueryString   string
	RemoteIP      string
	Headers       json.RawMessage
	Body          []byte
	BodyTruncated bool
	ContentLength int64
	CreatedAt     time.Time
}
