package app

import (
	"context"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type App struct {
	cfg   Config
	store *Store
}

func New(cfg Config, store *Store) *App {
	return &App{cfg: cfg, store: store}
}

func (a *App) RunCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Duration(a.cfg.CleanupInterval) * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := a.store.Cleanup(ctx, a.cfg.RetentionDays); err != nil {
				log.Printf("cleanup: %v", err)
			}
		}
	}
}

func (a *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")

	if r.URL.Path == "/healthz" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
		return
	}

	if strings.HasPrefix(r.URL.Path, "/api/") {
		a.handleAPI(w, r)
		return
	}

	if strings.HasPrefix(r.URL.Path, "/at/") && !a.isRequestDetailPage(r) {
		a.handleCapture(w, r)
		return
	}

	a.serveStatic(w, r)
}

func (a *App) isRequestDetailPage(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if !strings.Contains(r.Header.Get("Accept"), "text/html") {
		return false
	}
	parts := pathParts(r.URL.Path)
	return len(parts) == 3 && parts[0] == "at" && validateSlug(parts[1]) && validatePublicID(parts[2])
}

func (a *App) serveStatic(w http.ResponseWriter, r *http.Request) {
	staticRoot, err := filepath.Abs(a.cfg.StaticDir)
	if err != nil {
		http.Error(w, "static directory is not available", http.StatusInternalServerError)
		return
	}

	requested := strings.TrimPrefix(filepath.Clean("/"+r.URL.Path), "/")
	if requested != "." && requested != "" {
		candidate := filepath.Join(staticRoot, requested)
		if strings.HasPrefix(candidate, staticRoot+string(os.PathSeparator)) {
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				setStaticContentType(w, candidate)
				http.ServeFile(w, r, candidate)
				return
			}
		}
	}

	indexPath := filepath.Join(staticRoot, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		http.Error(w, "frontend build is missing; run npm run build in frontend or use the Docker image", http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, indexPath)
}

func setStaticContentType(w http.ResponseWriter, path string) {
	if contentType := mime.TypeByExtension(filepath.Ext(path)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
}

func (a *App) handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	parts := pathParts(r.URL.Path)
	if len(parts) < 2 || parts[0] != "api" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	switch parts[1] {
	case "telegram":
		a.handleTelegramAPI(w, r, parts[2:])
	case "session":
		a.handleSession(w, r)
	case "webhooks":
		a.handleOwnerAPI(w, r, parts[2:])
	case "share":
		a.handleShareAPI(w, r, parts[2:])
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (a *App) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	_, ownerToken, ok := a.ensureOwner(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"csrfToken":     a.csrfToken(ownerToken),
		"baseUrl":       a.baseURL(r),
		"retentionDays": a.cfg.RetentionDays,
		"maxBodyBytes":  a.cfg.MaxBodyBytes,
	})
}

func (a *App) handleOwnerAPI(w http.ResponseWriter, r *http.Request, parts []string) {
	owner, ownerToken, ok := a.ensureOwner(w, r)
	if !ok {
		return
	}
	if isMutating(r.Method) && !a.checkCSRF(w, r, ownerToken) {
		return
	}

	if len(parts) == 0 {
		switch r.Method {
		case http.MethodGet:
			webhooks, err := a.store.ListWebhooks(r.Context(), owner.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not load webhooks")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"webhooks": a.ownerWebhookResponses(r, webhooks),
			})
		case http.MethodPost:
			webhook, err := a.store.CreateWebhook(r.Context(), owner.ID)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not create webhook")
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{
				"webhook": a.ownerWebhookResponse(r, webhook),
			})
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	slug := parts[0]
	if !validateSlug(slug) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}

	if len(parts) == 1 {
		switch r.Method {
		case http.MethodGet:
			webhook, err := a.store.GetWebhookForOwner(r.Context(), owner.ID, slug)
			if err != nil {
				writeSQLError(w, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"webhook": a.ownerWebhookResponse(r, webhook),
			})
		case http.MethodDelete:
			deleted, err := a.store.DeleteWebhook(r.Context(), owner.ID, slug)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not delete webhook")
				return
			}
			if !deleted {
				writeError(w, http.StatusNotFound, "not found")
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	switch parts[1] {
	case "share":
		a.handleOwnerShare(w, r, owner.ID, slug)
	case "telegram":
		a.handleOwnerWebhookTelegram(w, r, owner.ID, slug)
	case "requests":
		a.handleOwnerRequests(w, r, owner.ID, slug, parts[2:])
	default:
		writeError(w, http.StatusNotFound, "not found")
	}
}

func (a *App) handleTelegramAPI(w http.ResponseWriter, r *http.Request, parts []string) {
	owner, ownerToken, ok := a.ensureOwner(w, r)
	if !ok {
		return
	}
	if isMutating(r.Method) && !a.checkCSRF(w, r, ownerToken) {
		return
	}

	if len(parts) == 0 {
		switch r.Method {
		case http.MethodGet:
			settings, err := a.store.GetTelegramSettings(r.Context(), owner.ID)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					writeJSON(w, http.StatusOK, map[string]any{
						"settings": telegramSettingsDTO{Configured: false},
					})
					return
				}
				writeError(w, http.StatusInternalServerError, "could not load telegram settings")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"settings": telegramSettingsResponse(settings),
			})
		case http.MethodPatch:
			settings, ok := a.telegramSettingsFromRequest(w, r, owner.ID)
			if !ok {
				return
			}
			saved, err := a.store.UpsertTelegramSettings(r.Context(), settings)
			if err != nil {
				writeError(w, http.StatusInternalServerError, "could not save telegram settings")
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{
				"settings": telegramSettingsResponse(saved),
			})
		default:
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		}
		return
	}

	if len(parts) == 1 && parts[0] == "test" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		settings, ok := a.telegramSettingsFromRequest(w, r, owner.ID)
		if !ok {
			return
		}
		if err := sendTelegramTest(settings); err != nil {
			logNotificationError("test telegram notification", err)
			writeError(w, http.StatusBadGateway, telegramTestErrorMessage(err))
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}

	writeError(w, http.StatusNotFound, "not found")
}

func (a *App) handleOwnerShare(w http.ResponseWriter, r *http.Request, ownerID int64, slug string) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	webhook, err := a.store.SetShareEnabled(r.Context(), ownerID, slug, body.Enabled)
	if err != nil {
		writeSQLError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"webhook": a.ownerWebhookResponse(r, webhook),
	})
}

func (a *App) handleOwnerWebhookTelegram(w http.ResponseWriter, r *http.Request, ownerID int64, slug string) {
	if r.Method != http.MethodPatch {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.Enabled {
		if _, err := a.store.GetTelegramSettings(r.Context(), ownerID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusConflict, "telegram settings required")
				return
			}
			writeError(w, http.StatusInternalServerError, "could not load telegram settings")
			return
		}
	}
	webhook, err := a.store.SetTelegramEnabled(r.Context(), ownerID, slug, body.Enabled)
	if err != nil {
		writeSQLError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"webhook": a.ownerWebhookResponse(r, webhook),
	})
}

func (a *App) telegramSettingsFromRequest(w http.ResponseWriter, r *http.Request, ownerID int64) (TelegramSettings, bool) {
	var body struct {
		BotToken      string `json:"botToken"`
		ChatID        string `json:"chatId"`
		ProxyEnabled  bool   `json:"proxyEnabled"`
		ProxyHost     string `json:"proxyHost"`
		ProxyPort     int    `json:"proxyPort"`
		ProxyUsername string `json:"proxyUsername"`
		ProxyPassword string `json:"proxyPassword"`
	}
	if err := readJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return TelegramSettings{}, false
	}

	existing, err := a.store.GetTelegramSettings(r.Context(), ownerID)
	hasExisting := err == nil
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusInternalServerError, "could not load telegram settings")
		return TelegramSettings{}, false
	}

	settings := TelegramSettings{
		OwnerID:       ownerID,
		BotToken:      strings.TrimSpace(body.BotToken),
		ChatID:        strings.TrimSpace(body.ChatID),
		ProxyEnabled:  body.ProxyEnabled,
		ProxyHost:     strings.TrimSpace(body.ProxyHost),
		ProxyPort:     body.ProxyPort,
		ProxyUsername: strings.TrimSpace(body.ProxyUsername),
		ProxyPassword: body.ProxyPassword,
	}
	if settings.BotToken == "" && hasExisting {
		settings.BotToken = existing.BotToken
	}
	if settings.ChatID == "" && hasExisting {
		settings.ChatID = existing.ChatID
	}

	if settings.BotToken == "" {
		writeError(w, http.StatusBadRequest, "telegram bot token is required")
		return TelegramSettings{}, false
	}
	if settings.ChatID == "" {
		writeError(w, http.StatusBadRequest, "telegram chat id is required")
		return TelegramSettings{}, false
	}
	if settings.ProxyEnabled {
		if settings.ProxyHost == "" {
			writeError(w, http.StatusBadRequest, "SOCKS5 proxy host is required")
			return TelegramSettings{}, false
		}
		if settings.ProxyPort < 1 || settings.ProxyPort > 65535 {
			writeError(w, http.StatusBadRequest, "SOCKS5 proxy port is invalid")
			return TelegramSettings{}, false
		}
		if settings.ProxyPassword == "" && shouldReuseTelegramProxyPassword(settings, existing, hasExisting) {
			settings.ProxyPassword = existing.ProxyPassword
		}
	} else {
		settings.ProxyHost = ""
		settings.ProxyPort = 0
		settings.ProxyUsername = ""
		settings.ProxyPassword = ""
	}

	return settings, true
}

func shouldReuseTelegramProxyPassword(settings TelegramSettings, existing TelegramSettings, hasExisting bool) bool {
	return hasExisting &&
		existing.ProxyEnabled &&
		existing.ProxyPassword != "" &&
		existing.ProxyHost == settings.ProxyHost &&
		existing.ProxyPort == settings.ProxyPort &&
		existing.ProxyUsername == settings.ProxyUsername
}

func (a *App) handleOwnerRequests(w http.ResponseWriter, r *http.Request, ownerID int64, slug string, parts []string) {
	webhook, err := a.store.GetWebhookForOwner(r.Context(), ownerID, slug)
	if err != nil {
		writeSQLError(w, err)
		return
	}
	if len(parts) == 0 {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		requests, err := a.store.ListRequests(r.Context(), webhook.ID, limitFromRequest(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load requests")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"requests": a.requestResponses(r, webhook, requests),
		})
		return
	}
	if len(parts) != 1 || !validatePublicID(parts[0]) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	if r.Method == http.MethodDelete {
		deleted, err := a.store.DeleteRequest(r.Context(), webhook.ID, parts[0])
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not delete request")
			return
		}
		if !deleted {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	request, err := a.store.GetRequest(r.Context(), webhook.ID, parts[0])
	if err != nil {
		writeSQLError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"request": a.requestResponse(r, webhook, request),
	})
}

func (a *App) handleShareAPI(w http.ResponseWriter, r *http.Request, parts []string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	if len(parts) == 0 || !validateSlug(parts[0]) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	webhook, ok := a.sharedWebhook(w, r, parts[0])
	if !ok {
		return
	}
	if len(parts) == 1 {
		requests, err := a.store.ListRequests(r.Context(), webhook.ID, limitFromRequest(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load requests")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"webhook":  a.webhookResponse(r, webhook),
			"requests": a.requestResponses(r, webhook, requests),
		})
		return
	}
	if len(parts) == 2 && parts[1] == "requests" {
		requests, err := a.store.ListRequests(r.Context(), webhook.ID, limitFromRequest(r))
		if err != nil {
			writeError(w, http.StatusInternalServerError, "could not load requests")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"requests": a.requestResponses(r, webhook, requests),
		})
		return
	}
	if len(parts) == 3 && parts[1] == "requests" && validatePublicID(parts[2]) {
		request, err := a.store.GetRequest(r.Context(), webhook.ID, parts[2])
		if err != nil {
			writeSQLError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"request": a.requestResponse(r, webhook, request),
		})
		return
	}
	writeError(w, http.StatusNotFound, "not found")
}

func (a *App) sharedWebhook(w http.ResponseWriter, r *http.Request, slug string) (Webhook, bool) {
	webhook, err := a.store.GetWebhookBySlug(r.Context(), slug)
	if err != nil {
		writeSQLError(w, err)
		return Webhook{}, false
	}
	if !webhook.ShareEnabled {
		writeError(w, http.StatusNotFound, "not found")
		return Webhook{}, false
	}
	token := r.URL.Query().Get("id")
	if token == "" || !constantEqual(token, a.shareToken(webhook)) {
		writeError(w, http.StatusForbidden, "invalid share token")
		return Webhook{}, false
	}
	return webhook, true
}

func (a *App) handleCapture(w http.ResponseWriter, r *http.Request) {
	slug, ok := captureSlug(r.URL.Path)
	if !ok {
		writeCapturePlain(w, http.StatusNotFound, false)
		return
	}
	webhook, err := a.store.GetWebhookBySlug(r.Context(), slug)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeCapturePlain(w, http.StatusNotFound, false)
			return
		}
		writeCapturePlain(w, http.StatusInternalServerError, false)
		return
	}

	body, truncated, err := readLimitedBody(r.Body, a.cfg.MaxBodyBytes)
	if err != nil {
		writeCapturePlain(w, http.StatusBadRequest, false)
		return
	}

	publicID, err := randomBase62(32)
	if err != nil {
		writeCapturePlain(w, http.StatusInternalServerError, false)
		return
	}
	for i := 0; i < 5; i++ {
		request, err := a.store.CreateRequest(r.Context(), RequestInput{
			WebhookID:     webhook.ID,
			PublicID:      publicID,
			Method:        r.Method,
			Path:          r.URL.Path,
			QueryString:   r.URL.RawQuery,
			RemoteIP:      a.clientIP(r),
			Headers:       cloneHeaders(r.Header),
			Body:          body,
			BodyTruncated: truncated,
			ContentLength: contentLength(r, body, truncated),
		})
		if err == nil {
			if webhook.TelegramEnabled {
				a.notifyTelegramAsync(webhook, request, r.Host, a.requestDetailURL(r, webhook, request))
			}
			a.writeCaptureResponse(w, r)
			return
		}
		publicID, err = randomBase62(32)
		if err != nil {
			break
		}
	}
	writeCapturePlain(w, http.StatusInternalServerError, false)
}

func (a *App) writeCaptureResponse(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodHead {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(a.cfg.CaptureStatusCode)
		return
	}
	writeCapturePlain(w, a.cfg.CaptureStatusCode, a.cfg.CaptureStatusCode >= 200 && a.cfg.CaptureStatusCode < 300)
}

func writeCapturePlain(w http.ResponseWriter, status int, ok bool) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	if ok {
		_, _ = w.Write([]byte("ok\n"))
		return
	}
	_, _ = w.Write([]byte("error\n"))
}

func (a *App) ensureOwner(w http.ResponseWriter, r *http.Request) (Owner, string, bool) {
	var token string
	if cookie, err := r.Cookie(a.cfg.CookieName); err == nil {
		token = cookie.Value
	}
	if err := decodeOwnerToken(token); err != nil {
		generated, genErr := randomToken(32)
		if genErr != nil {
			writeError(w, http.StatusInternalServerError, "could not create owner token")
			return Owner{}, "", false
		}
		token = generated
		a.setOwnerCookie(w, token)
	}

	owner, err := a.store.EnsureOwner(r.Context(), tokenHash(token))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not load owner")
		return Owner{}, "", false
	}
	return owner, token, true
}

func (a *App) setOwnerCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     a.cfg.CookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int((365 * 24 * time.Hour).Seconds()),
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
	})
}

func (a *App) csrfToken(ownerToken string) string {
	return hmacHex(a.cfg.AppSecret, "csrf", ownerToken)
}

func (a *App) checkCSRF(w http.ResponseWriter, r *http.Request, ownerToken string) bool {
	token := r.Header.Get("X-CSRF-Token")
	if token == "" || !constantEqual(token, a.csrfToken(ownerToken)) {
		writeError(w, http.StatusForbidden, "invalid CSRF token")
		return false
	}
	return true
}

func (a *App) shareToken(webhook Webhook) string {
	token := hmacHex(a.cfg.AppSecret, "share", webhook.ShareNonce, webhook.Slug)
	return token[:32]
}

func (a *App) webhookResponses(r *http.Request, webhooks []Webhook) []webhookDTO {
	out := make([]webhookDTO, 0, len(webhooks))
	for _, webhook := range webhooks {
		out = append(out, a.webhookResponse(r, webhook))
	}
	return out
}

func (a *App) ownerWebhookResponses(r *http.Request, webhooks []Webhook) []webhookDTO {
	out := make([]webhookDTO, 0, len(webhooks))
	for _, webhook := range webhooks {
		out = append(out, a.ownerWebhookResponse(r, webhook))
	}
	return out
}

func (a *App) webhookResponse(r *http.Request, webhook Webhook) webhookDTO {
	base := a.baseURL(r)
	dto := webhookDTO{
		Slug:          webhook.Slug,
		URL:           fmt.Sprintf("%s/at/%s", base, webhook.Slug),
		ShareEnabled:  webhook.ShareEnabled,
		CreatedAt:     webhook.CreatedAt,
		UpdatedAt:     webhook.UpdatedAt,
		RequestCount:  webhook.RequestCount,
		LastRequestAt: webhook.LastRequestAt,
	}
	if webhook.ShareEnabled {
		shareURL := fmt.Sprintf("%s/share/%s?id=%s", base, webhook.Slug, a.shareToken(webhook))
		dto.ShareURL = &shareURL
	}
	return dto
}

func (a *App) ownerWebhookResponse(r *http.Request, webhook Webhook) webhookDTO {
	dto := a.webhookResponse(r, webhook)
	dto.TelegramEnabled = &webhook.TelegramEnabled
	return dto
}

func telegramSettingsResponse(settings TelegramSettings) telegramSettingsDTO {
	return telegramSettingsDTO{
		Configured:              settings.BotToken != "" && settings.ChatID != "",
		ChatID:                  settings.ChatID,
		ProxyEnabled:            settings.ProxyEnabled,
		ProxyHost:               settings.ProxyHost,
		ProxyPort:               settings.ProxyPort,
		ProxyUsername:           settings.ProxyUsername,
		ProxyPasswordConfigured: settings.ProxyPassword != "",
	}
}

func telegramTestErrorMessage(err error) string {
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "timeout") || strings.Contains(message, "deadline") || strings.Contains(message, "timed out") {
		return "telegram test timed out after 10 seconds"
	}
	if strings.Contains(message, "proxy") || strings.Contains(message, "socks") {
		return "telegram test failed; check SOCKS5 proxy settings"
	}
	return "telegram test failed; check bot token, chat id, and proxy settings"
}

func (a *App) requestResponses(r *http.Request, webhook Webhook, requests []CapturedRequest) []requestDTO {
	out := make([]requestDTO, 0, len(requests))
	for _, request := range requests {
		out = append(out, a.requestResponse(r, webhook, request))
	}
	return out
}

func (a *App) requestResponse(r *http.Request, webhook Webhook, request CapturedRequest) requestDTO {
	var headers map[string][]string
	_ = json.Unmarshal(request.Headers, &headers)
	bodyText, bodyBase64, encoding := encodeBody(request.Body)
	target := request.Path
	if request.QueryString != "" {
		target += "?" + request.QueryString
	}
	return requestDTO{
		ID:            request.PublicID,
		Method:        request.Method,
		Path:          request.Path,
		QueryString:   request.QueryString,
		Target:        target,
		RemoteIP:      request.RemoteIP,
		Headers:       headers,
		BodyText:      bodyText,
		BodyBase64:    bodyBase64,
		BodyEncoding:  encoding,
		BodySize:      len(request.Body),
		BodyTruncated: request.BodyTruncated,
		ContentLength: request.ContentLength,
		CreatedAt:     request.CreatedAt,
		DetailURL:     a.requestDetailURL(r, webhook, request),
		ShareURL:      fmt.Sprintf("%s/share/%s/%s?id=%s", a.baseURL(r), webhook.Slug, request.PublicID, a.shareToken(webhook)),
	}
}

func (a *App) requestDetailURL(r *http.Request, webhook Webhook, request CapturedRequest) string {
	return fmt.Sprintf("%s/at/%s/%s", a.baseURL(r), webhook.Slug, request.PublicID)
}

func (a *App) baseURL(r *http.Request) string {
	if a.cfg.PublicBaseURL != "" {
		return a.cfg.PublicBaseURL
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if a.cfg.TrustProxy {
		if forwardedProto := firstHeaderValue(r.Header.Get("X-Forwarded-Proto")); forwardedProto == "http" || forwardedProto == "https" {
			scheme = forwardedProto
		}
	}
	host := r.Host
	if a.cfg.TrustProxy {
		if forwardedHost := firstHeaderValue(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
			host = forwardedHost
		}
	}
	return scheme + "://" + host
}

func (a *App) clientIP(r *http.Request) string {
	if a.cfg.TrustProxy {
		if forwarded := firstHeaderValue(r.Header.Get("X-Forwarded-For")); forwarded != "" {
			return forwarded
		}
		if realIP := firstHeaderValue(r.Header.Get("X-Real-IP")); realIP != "" {
			return realIP
		}
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

type webhookDTO struct {
	Slug            string     `json:"slug"`
	URL             string     `json:"url"`
	ShareEnabled    bool       `json:"shareEnabled"`
	ShareURL        *string    `json:"shareUrl"`
	TelegramEnabled *bool      `json:"telegramEnabled,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	RequestCount    int64      `json:"requestCount"`
	LastRequestAt   *time.Time `json:"lastRequestAt"`
}

type telegramSettingsDTO struct {
	Configured              bool   `json:"configured"`
	ChatID                  string `json:"chatId"`
	ProxyEnabled            bool   `json:"proxyEnabled"`
	ProxyHost               string `json:"proxyHost"`
	ProxyPort               int    `json:"proxyPort"`
	ProxyUsername           string `json:"proxyUsername"`
	ProxyPasswordConfigured bool   `json:"proxyPasswordConfigured"`
}

type requestDTO struct {
	ID            string              `json:"id"`
	Method        string              `json:"method"`
	Path          string              `json:"path"`
	QueryString   string              `json:"queryString"`
	Target        string              `json:"target"`
	RemoteIP      string              `json:"remoteIp"`
	Headers       map[string][]string `json:"headers"`
	BodyText      string              `json:"bodyText"`
	BodyBase64    string              `json:"bodyBase64"`
	BodyEncoding  string              `json:"bodyEncoding"`
	BodySize      int                 `json:"bodySize"`
	BodyTruncated bool                `json:"bodyTruncated"`
	ContentLength int64               `json:"contentLength"`
	CreatedAt     time.Time           `json:"createdAt"`
	DetailURL     string              `json:"detailUrl"`
	ShareURL      string              `json:"shareUrl"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func writeSQLError(w http.ResponseWriter, err error) {
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	writeError(w, http.StatusInternalServerError, "database error")
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func readLimitedBody(body io.ReadCloser, max int64) ([]byte, bool, error) {
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, max+1))
	if err != nil {
		return nil, false, err
	}
	if int64(len(data)) > max {
		return data[:max], true, nil
	}
	return data, false, nil
}

func encodeBody(body []byte) (text string, b64 string, encoding string) {
	if len(body) == 0 {
		return "", "", "text"
	}
	if isMostlyText(body) {
		return string(body), "", "text"
	}
	return "", base64.StdEncoding.EncodeToString(body), "base64"
}

func isMostlyText(body []byte) bool {
	if !utf8.Valid(body) {
		return false
	}
	for _, b := range body {
		if b == 0 {
			return false
		}
		if b < 32 && b != '\n' && b != '\r' && b != '\t' {
			return false
		}
	}
	return true
}

func cloneHeaders(headers http.Header) map[string][]string {
	out := make(map[string][]string, len(headers))
	for key, values := range headers {
		copied := make([]string, len(values))
		copy(copied, values)
		out[key] = copied
	}
	return out
}

func pathParts(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

func captureSlug(path string) (string, bool) {
	parts := pathParts(path)
	if len(parts) < 2 || parts[0] != "at" || !validateSlug(parts[1]) {
		return "", false
	}
	return parts[1], true
}

func validateSlug(slug string) bool {
	if len(slug) < 5 || len(slug) > 80 {
		return false
	}
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

func isMutating(method string) bool {
	return method == http.MethodPost || method == http.MethodPatch || method == http.MethodPut || method == http.MethodDelete
}

func limitFromRequest(r *http.Request) int {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil || limit < 1 || limit > 200 {
		return 100
	}
	return limit
}

func firstHeaderValue(value string) string {
	if value == "" {
		return ""
	}
	first := strings.TrimSpace(strings.Split(value, ",")[0])
	if strings.ContainsAny(first, "\r\n") {
		return ""
	}
	return first
}

func contentLength(r *http.Request, body []byte, truncated bool) int64 {
	if r.ContentLength >= 0 {
		return r.ContentLength
	}
	if truncated {
		return -1
	}
	return int64(len(body))
}
