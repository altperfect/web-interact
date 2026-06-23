package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type Store struct {
	db *sql.DB
}

func OpenStore(ctx context.Context, databaseURL string) (*Store, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	deadline := time.Now().Add(45 * time.Second)
	for {
		err = db.PingContext(ctx)
		if err == nil {
			return &Store{db: db}, nil
		}
		if time.Now().After(deadline) {
			db.Close()
			return nil, err
		}
		select {
		case <-ctx.Done():
			db.Close()
			return nil, ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Migrate(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS owners (
	id BIGSERIAL PRIMARY KEY,
	token_hash TEXT NOT NULL UNIQUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS webhooks (
	id BIGSERIAL PRIMARY KEY,
	slug TEXT NOT NULL UNIQUE,
	owner_id BIGINT NOT NULL REFERENCES owners(id) ON DELETE CASCADE,
	share_enabled BOOLEAN NOT NULL DEFAULT false,
	telegram_enabled BOOLEAN NOT NULL DEFAULT false,
	share_nonce TEXT NOT NULL,
	response_body TEXT NOT NULL DEFAULT E'ok\n',
	response_content_type TEXT NOT NULL DEFAULT 'text/plain; charset=utf-8',
	response_status_code INTEGER NOT NULL DEFAULT 200,
	response_location TEXT NOT NULL DEFAULT '',
	response_headers JSONB NOT NULL DEFAULT '[]'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE webhooks ADD COLUMN IF NOT EXISTS telegram_enabled BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE webhooks ADD COLUMN IF NOT EXISTS response_body TEXT NOT NULL DEFAULT E'ok\n';
ALTER TABLE webhooks ADD COLUMN IF NOT EXISTS response_content_type TEXT NOT NULL DEFAULT 'text/plain; charset=utf-8';
ALTER TABLE webhooks ADD COLUMN IF NOT EXISTS response_status_code INTEGER NOT NULL DEFAULT 200;
ALTER TABLE webhooks ADD COLUMN IF NOT EXISTS response_location TEXT NOT NULL DEFAULT '';
ALTER TABLE webhooks ADD COLUMN IF NOT EXISTS response_headers JSONB NOT NULL DEFAULT '[]'::jsonb;

CREATE INDEX IF NOT EXISTS webhooks_owner_id_idx ON webhooks(owner_id);

CREATE TABLE IF NOT EXISTS telegram_settings (
	owner_id BIGINT PRIMARY KEY REFERENCES owners(id) ON DELETE CASCADE,
	bot_token TEXT NOT NULL,
	chat_id TEXT NOT NULL,
	proxy_enabled BOOLEAN NOT NULL DEFAULT false,
	proxy_host TEXT NOT NULL DEFAULT '',
	proxy_port INTEGER NOT NULL DEFAULT 0,
	proxy_username TEXT NOT NULL DEFAULT '',
	proxy_password TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS captured_requests (
	id BIGSERIAL PRIMARY KEY,
	webhook_id BIGINT NOT NULL REFERENCES webhooks(id) ON DELETE CASCADE,
	public_id TEXT NOT NULL UNIQUE,
	method TEXT NOT NULL,
	path TEXT NOT NULL,
	query_string TEXT NOT NULL,
	remote_ip TEXT NOT NULL,
	headers JSONB NOT NULL,
	body BYTEA NOT NULL,
	body_truncated BOOLEAN NOT NULL,
	content_length BIGINT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS captured_requests_webhook_created_idx ON captured_requests(webhook_id, created_at DESC);
CREATE INDEX IF NOT EXISTS captured_requests_created_idx ON captured_requests(created_at);
`
	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *Store) Cleanup(ctx context.Context, retentionDays int) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM captured_requests WHERE created_at < now() - ($1::int * interval '1 day')`, retentionDays)
	return err
}

func (s *Store) EnsureOwner(ctx context.Context, hash string) (Owner, error) {
	var owner Owner
	err := s.db.QueryRowContext(ctx, `
INSERT INTO owners (token_hash)
VALUES ($1)
ON CONFLICT (token_hash) DO UPDATE SET token_hash = EXCLUDED.token_hash
RETURNING id, token_hash, created_at
`, hash).Scan(&owner.ID, &owner.TokenHash, &owner.CreatedAt)
	return owner, err
}

func (s *Store) CreateWebhook(ctx context.Context, ownerID int64) (Webhook, error) {
	for i := 0; i < 20; i++ {
		slug, err := generateSlug()
		if err != nil {
			return Webhook{}, err
		}
		nonce, err := randomToken(18)
		if err != nil {
			return Webhook{}, err
		}
		var webhook Webhook
		err = s.db.QueryRowContext(ctx, `
INSERT INTO webhooks (slug, owner_id, share_nonce)
VALUES ($1, $2, $3)
ON CONFLICT (slug) DO NOTHING
RETURNING id, slug, owner_id, share_enabled, telegram_enabled, share_nonce, response_body, response_content_type, response_status_code, response_location, response_headers, created_at, updated_at
`, slug, ownerID, nonce).Scan(
			&webhook.ID,
			&webhook.Slug,
			&webhook.OwnerID,
			&webhook.ShareEnabled,
			&webhook.TelegramEnabled,
			&webhook.ShareNonce,
			&webhook.ResponseBody,
			&webhook.ResponseType,
			&webhook.ResponseStatus,
			&webhook.ResponseLocation,
			&webhook.ResponseHeaders,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
		)
		if err == nil {
			return webhook, nil
		}
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		return Webhook{}, err
	}
	return Webhook{}, errors.New("could not generate a unique webhook slug")
}

func (s *Store) ListWebhooks(ctx context.Context, ownerID int64) ([]Webhook, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
	w.id,
	w.slug,
	w.owner_id,
	w.share_enabled,
	w.telegram_enabled,
	w.share_nonce,
	w.response_body,
	w.response_content_type,
	w.response_status_code,
	w.response_location,
	w.response_headers,
	w.created_at,
	w.updated_at,
	COUNT(r.id)::bigint AS request_count,
	MAX(r.created_at) AS last_request_at
FROM webhooks w
LEFT JOIN captured_requests r ON r.webhook_id = w.id
WHERE w.owner_id = $1
GROUP BY w.id
ORDER BY w.created_at DESC
`, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var webhooks []Webhook
	for rows.Next() {
		var webhook Webhook
		var last sql.NullTime
		if err := rows.Scan(
			&webhook.ID,
			&webhook.Slug,
			&webhook.OwnerID,
			&webhook.ShareEnabled,
			&webhook.TelegramEnabled,
			&webhook.ShareNonce,
			&webhook.ResponseBody,
			&webhook.ResponseType,
			&webhook.ResponseStatus,
			&webhook.ResponseLocation,
			&webhook.ResponseHeaders,
			&webhook.CreatedAt,
			&webhook.UpdatedAt,
			&webhook.RequestCount,
			&last,
		); err != nil {
			return nil, err
		}
		if last.Valid {
			webhook.LastRequestAt = &last.Time
		}
		webhooks = append(webhooks, webhook)
	}
	return webhooks, rows.Err()
}

func (s *Store) GetWebhookForOwner(ctx context.Context, ownerID int64, slug string) (Webhook, error) {
	return s.getWebhook(ctx, `WHERE owner_id = $1 AND slug = $2`, ownerID, slug)
}

func (s *Store) GetWebhookBySlug(ctx context.Context, slug string) (Webhook, error) {
	return s.getWebhook(ctx, `WHERE slug = $1`, slug)
}

func (s *Store) getWebhook(ctx context.Context, where string, args ...any) (Webhook, error) {
	query := fmt.Sprintf(`
SELECT
	w.id,
	w.slug,
	w.owner_id,
	w.share_enabled,
	w.telegram_enabled,
	w.share_nonce,
	w.response_body,
	w.response_content_type,
	w.response_status_code,
	w.response_location,
	w.response_headers,
	w.created_at,
	w.updated_at,
	(SELECT COUNT(*)::bigint FROM captured_requests r WHERE r.webhook_id = w.id) AS request_count,
	(SELECT MAX(created_at) FROM captured_requests r WHERE r.webhook_id = w.id) AS last_request_at
FROM webhooks w
%s
`, where)
	var webhook Webhook
	var last sql.NullTime
	err := s.db.QueryRowContext(ctx, query, args...).Scan(
		&webhook.ID,
		&webhook.Slug,
		&webhook.OwnerID,
		&webhook.ShareEnabled,
		&webhook.TelegramEnabled,
		&webhook.ShareNonce,
		&webhook.ResponseBody,
		&webhook.ResponseType,
		&webhook.ResponseStatus,
		&webhook.ResponseLocation,
		&webhook.ResponseHeaders,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
		&webhook.RequestCount,
		&last,
	)
	if last.Valid {
		webhook.LastRequestAt = &last.Time
	}
	return webhook, err
}

func (s *Store) DeleteWebhook(ctx context.Context, ownerID int64, slug string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM webhooks WHERE owner_id = $1 AND slug = $2`, ownerID, slug)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func (s *Store) SetShareEnabled(ctx context.Context, ownerID int64, slug string, enabled bool) (Webhook, error) {
	var webhook Webhook
	err := s.db.QueryRowContext(ctx, `
UPDATE webhooks
SET share_enabled = $3, updated_at = now()
WHERE owner_id = $1 AND slug = $2
RETURNING id, slug, owner_id, share_enabled, telegram_enabled, share_nonce, response_body, response_content_type, response_status_code, response_location, response_headers, created_at, updated_at
`, ownerID, slug, enabled).Scan(
		&webhook.ID,
		&webhook.Slug,
		&webhook.OwnerID,
		&webhook.ShareEnabled,
		&webhook.TelegramEnabled,
		&webhook.ShareNonce,
		&webhook.ResponseBody,
		&webhook.ResponseType,
		&webhook.ResponseStatus,
		&webhook.ResponseLocation,
		&webhook.ResponseHeaders,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
	)
	return webhook, err
}

func (s *Store) SetTelegramEnabled(ctx context.Context, ownerID int64, slug string, enabled bool) (Webhook, error) {
	var webhook Webhook
	err := s.db.QueryRowContext(ctx, `
UPDATE webhooks
SET telegram_enabled = $3, updated_at = now()
WHERE owner_id = $1 AND slug = $2
RETURNING id, slug, owner_id, share_enabled, telegram_enabled, share_nonce, response_body, response_content_type, response_status_code, response_location, response_headers, created_at, updated_at
`, ownerID, slug, enabled).Scan(
		&webhook.ID,
		&webhook.Slug,
		&webhook.OwnerID,
		&webhook.ShareEnabled,
		&webhook.TelegramEnabled,
		&webhook.ShareNonce,
		&webhook.ResponseBody,
		&webhook.ResponseType,
		&webhook.ResponseStatus,
		&webhook.ResponseLocation,
		&webhook.ResponseHeaders,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
	)
	return webhook, err
}

func (s *Store) SetWebhookResponse(ctx context.Context, ownerID int64, slug string, body string, contentType string, statusCode int, location string, headers []ResponseHeader) (Webhook, error) {
	encodedHeaders, err := json.Marshal(headers)
	if err != nil {
		return Webhook{}, err
	}
	var webhook Webhook
	err = s.db.QueryRowContext(ctx, `
UPDATE webhooks
SET response_body = $3,
	response_content_type = $4,
	response_status_code = $5,
	response_location = $6,
	response_headers = $7::jsonb,
	updated_at = now()
WHERE owner_id = $1 AND slug = $2
RETURNING id, slug, owner_id, share_enabled, telegram_enabled, share_nonce, response_body, response_content_type, response_status_code, response_location, response_headers, created_at, updated_at
`, ownerID, slug, body, contentType, statusCode, location, encodedHeaders).Scan(
		&webhook.ID,
		&webhook.Slug,
		&webhook.OwnerID,
		&webhook.ShareEnabled,
		&webhook.TelegramEnabled,
		&webhook.ShareNonce,
		&webhook.ResponseBody,
		&webhook.ResponseType,
		&webhook.ResponseStatus,
		&webhook.ResponseLocation,
		&webhook.ResponseHeaders,
		&webhook.CreatedAt,
		&webhook.UpdatedAt,
	)
	return webhook, err
}

func (s *Store) GetTelegramSettings(ctx context.Context, ownerID int64) (TelegramSettings, error) {
	var settings TelegramSettings
	err := s.db.QueryRowContext(ctx, `
SELECT owner_id, bot_token, chat_id, proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password, created_at, updated_at
FROM telegram_settings
WHERE owner_id = $1
`, ownerID).Scan(
		&settings.OwnerID,
		&settings.BotToken,
		&settings.ChatID,
		&settings.ProxyEnabled,
		&settings.ProxyHost,
		&settings.ProxyPort,
		&settings.ProxyUsername,
		&settings.ProxyPassword,
		&settings.CreatedAt,
		&settings.UpdatedAt,
	)
	return settings, err
}

func (s *Store) UpsertTelegramSettings(ctx context.Context, settings TelegramSettings) (TelegramSettings, error) {
	var saved TelegramSettings
	err := s.db.QueryRowContext(ctx, `
INSERT INTO telegram_settings (
	owner_id,
	bot_token,
	chat_id,
	proxy_enabled,
	proxy_host,
	proxy_port,
	proxy_username,
	proxy_password
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT (owner_id) DO UPDATE SET
	bot_token = EXCLUDED.bot_token,
	chat_id = EXCLUDED.chat_id,
	proxy_enabled = EXCLUDED.proxy_enabled,
	proxy_host = EXCLUDED.proxy_host,
	proxy_port = EXCLUDED.proxy_port,
	proxy_username = EXCLUDED.proxy_username,
	proxy_password = EXCLUDED.proxy_password,
	updated_at = now()
RETURNING owner_id, bot_token, chat_id, proxy_enabled, proxy_host, proxy_port, proxy_username, proxy_password, created_at, updated_at
`,
		settings.OwnerID,
		settings.BotToken,
		settings.ChatID,
		settings.ProxyEnabled,
		settings.ProxyHost,
		settings.ProxyPort,
		settings.ProxyUsername,
		settings.ProxyPassword,
	).Scan(
		&saved.OwnerID,
		&saved.BotToken,
		&saved.ChatID,
		&saved.ProxyEnabled,
		&saved.ProxyHost,
		&saved.ProxyPort,
		&saved.ProxyUsername,
		&saved.ProxyPassword,
		&saved.CreatedAt,
		&saved.UpdatedAt,
	)
	return saved, err
}

type RequestInput struct {
	WebhookID     int64
	PublicID      string
	Method        string
	Path          string
	QueryString   string
	RemoteIP      string
	Headers       map[string][]string
	Body          []byte
	BodyTruncated bool
	ContentLength int64
}

type RequestSearchResult struct {
	Webhook Webhook
	Request CapturedRequest
}

func (s *Store) CreateRequest(ctx context.Context, input RequestInput) (CapturedRequest, error) {
	headers, err := json.Marshal(input.Headers)
	if err != nil {
		return CapturedRequest{}, err
	}
	var request CapturedRequest
	err = s.db.QueryRowContext(ctx, `
INSERT INTO captured_requests (
	webhook_id,
	public_id,
	method,
	path,
	query_string,
	remote_ip,
	headers,
	body,
	body_truncated,
	content_length
)
VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9, $10)
RETURNING id, webhook_id, public_id, method, path, query_string, remote_ip, headers, body, body_truncated, content_length, created_at
`,
		input.WebhookID,
		input.PublicID,
		input.Method,
		input.Path,
		input.QueryString,
		input.RemoteIP,
		headers,
		input.Body,
		input.BodyTruncated,
		input.ContentLength,
	).Scan(
		&request.ID,
		&request.WebhookID,
		&request.PublicID,
		&request.Method,
		&request.Path,
		&request.QueryString,
		&request.RemoteIP,
		&request.Headers,
		&request.Body,
		&request.BodyTruncated,
		&request.ContentLength,
		&request.CreatedAt,
	)
	return request, err
}

func (s *Store) ListRequests(ctx context.Context, webhookID int64, limit int) ([]CapturedRequest, error) {
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, webhook_id, public_id, method, path, query_string, remote_ip, headers, body, body_truncated, content_length, created_at
FROM captured_requests
WHERE webhook_id = $1
ORDER BY created_at DESC
LIMIT $2
`, webhookID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var requests []CapturedRequest
	for rows.Next() {
		var request CapturedRequest
		if err := rows.Scan(
			&request.ID,
			&request.WebhookID,
			&request.PublicID,
			&request.Method,
			&request.Path,
			&request.QueryString,
			&request.RemoteIP,
			&request.Headers,
			&request.Body,
			&request.BodyTruncated,
			&request.ContentLength,
			&request.CreatedAt,
		); err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	return requests, rows.Err()
}

func (s *Store) SearchRequests(ctx context.Context, ownerID int64, searchQuery string, webhookSlug string, limit int) ([]RequestSearchResult, error) {
	if limit < 1 || limit > 100 {
		limit = 40
	}

	needle := "%" + escapeLikePattern(strings.ToLower(strings.TrimSpace(searchQuery))) + "%"
	args := []any{ownerID, needle}
	query := `
SELECT
	w.id,
	w.slug,
	w.owner_id,
	w.share_enabled,
	w.telegram_enabled,
	w.share_nonce,
	w.response_body,
	w.response_content_type,
	w.response_status_code,
	w.response_location,
	w.response_headers,
	w.created_at,
	w.updated_at,
	(SELECT COUNT(*)::bigint FROM captured_requests cr WHERE cr.webhook_id = w.id) AS request_count,
	(SELECT MAX(created_at) FROM captured_requests cr WHERE cr.webhook_id = w.id) AS last_request_at,
	r.id,
	r.webhook_id,
	r.public_id,
	r.method,
	r.path,
	r.query_string,
	r.remote_ip,
	r.headers,
	r.body,
	r.body_truncated,
	r.content_length,
	r.created_at
FROM captured_requests r
JOIN webhooks w ON w.id = r.webhook_id
WHERE w.owner_id = $1
AND (
	lower(r.public_id) LIKE $2 ESCAPE '\'
	OR lower(r.method) LIKE $2 ESCAPE '\'
	OR lower(r.path) LIKE $2 ESCAPE '\'
	OR lower(r.query_string) LIKE $2 ESCAPE '\'
	OR lower(r.remote_ip) LIKE $2 ESCAPE '\'
	OR lower(r.headers::text) LIKE $2 ESCAPE '\'
	OR lower(encode(r.body, 'escape')) LIKE $2 ESCAPE '\'
)`
	if webhookSlug != "" {
		args = append(args, webhookSlug)
		query += fmt.Sprintf("\nAND w.slug = $%d", len(args))
	}
	args = append(args, limit)
	query += fmt.Sprintf("\nORDER BY r.created_at DESC\nLIMIT $%d", len(args))

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RequestSearchResult
	for rows.Next() {
		var result RequestSearchResult
		var last sql.NullTime
		if err := rows.Scan(
			&result.Webhook.ID,
			&result.Webhook.Slug,
			&result.Webhook.OwnerID,
			&result.Webhook.ShareEnabled,
			&result.Webhook.TelegramEnabled,
			&result.Webhook.ShareNonce,
			&result.Webhook.ResponseBody,
			&result.Webhook.ResponseType,
			&result.Webhook.ResponseStatus,
			&result.Webhook.ResponseLocation,
			&result.Webhook.ResponseHeaders,
			&result.Webhook.CreatedAt,
			&result.Webhook.UpdatedAt,
			&result.Webhook.RequestCount,
			&last,
			&result.Request.ID,
			&result.Request.WebhookID,
			&result.Request.PublicID,
			&result.Request.Method,
			&result.Request.Path,
			&result.Request.QueryString,
			&result.Request.RemoteIP,
			&result.Request.Headers,
			&result.Request.Body,
			&result.Request.BodyTruncated,
			&result.Request.ContentLength,
			&result.Request.CreatedAt,
		); err != nil {
			return nil, err
		}
		if last.Valid {
			result.Webhook.LastRequestAt = &last.Time
		}
		results = append(results, result)
	}
	return results, rows.Err()
}

func (s *Store) GetRequest(ctx context.Context, webhookID int64, publicID string) (CapturedRequest, error) {
	var request CapturedRequest
	err := s.db.QueryRowContext(ctx, `
SELECT id, webhook_id, public_id, method, path, query_string, remote_ip, headers, body, body_truncated, content_length, created_at
FROM captured_requests
WHERE webhook_id = $1 AND public_id = $2
`, webhookID, publicID).Scan(
		&request.ID,
		&request.WebhookID,
		&request.PublicID,
		&request.Method,
		&request.Path,
		&request.QueryString,
		&request.RemoteIP,
		&request.Headers,
		&request.Body,
		&request.BodyTruncated,
		&request.ContentLength,
		&request.CreatedAt,
	)
	return request, err
}

func (s *Store) DeleteRequest(ctx context.Context, webhookID int64, publicID string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM captured_requests WHERE webhook_id = $1 AND public_id = $2`, webhookID, publicID)
	if err != nil {
		return false, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return affected > 0, nil
}

func escapeLikePattern(value string) string {
	replacer := strings.NewReplacer(
		`\`, `\\`,
		`%`, `\%`,
		`_`, `\_`,
	)
	return replacer.Replace(value)
}
