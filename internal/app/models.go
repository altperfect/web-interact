package app

import (
	"encoding/json"
	"time"
)

const defaultWebhookResponseBody = "ok\n"
const defaultWebhookResponseType = "text/plain; charset=utf-8"
const defaultWebhookResponseStatus = 200

type Owner struct {
	ID        int64
	TokenHash string
	CreatedAt time.Time
}

type Webhook struct {
	ID               int64
	Slug             string
	Name             string
	OwnerID          int64
	ShareEnabled     bool
	TelegramEnabled  bool
	ShareNonce       string
	ResponseBody     string
	ResponseType     string
	ResponseStatus   int
	ResponseLocation string
	ResponseHeaders  json.RawMessage
	CreatedAt        time.Time
	UpdatedAt        time.Time
	RequestCount     int64
	LastRequestAt    *time.Time
}

type ResponseHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
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
	Note          string
	CreatedAt     time.Time
}
