package app

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/containrrr/shoutrrr/pkg/util/jsonclient"
	"github.com/projectdiscovery/notify/pkg/providers/telegram"
)

const telegramMessageLimit = 3600
const telegramSendTimeout = 10 * time.Second

var notifyHTTPMu sync.Mutex

func (a *App) notifyTelegramAsync(webhook Webhook, request CapturedRequest, host string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()

		settings, err := a.store.GetTelegramSettings(ctx, webhook.OwnerID)
		if err != nil {
			if ctx.Err() == nil {
				logNotificationError("load telegram settings", err)
			}
			return
		}

		messages := buildTelegramRequestMessages(webhook, request, host)
		for _, message := range messages {
			if err := sendTelegramMessage(settings, message); err != nil {
				logNotificationError("send telegram notification", err)
				return
			}
		}
	}()
}

func sendTelegramTest(settings TelegramSettings) error {
	return sendTelegramMessage(settings, "Webhook Console test notification")
}

func sendTelegramMessage(settings TelegramSettings, message string) error {
	provider, err := telegram.New([]*telegram.Options{
		{
			ID:                "webhook-console",
			TelegramAPIKey:    settings.BotToken,
			TelegramChatID:    settings.ChatID,
			TelegramFormat:    "{{data}}",
			TelegramParseMode: "Markdown",
		},
	}, []string{"webhook-console"})
	if err != nil {
		return err
	}

	client, err := telegramHTTPClient(settings)
	if err != nil {
		return err
	}

	notifyHTTPMu.Lock()
	defer notifyHTTPMu.Unlock()

	previousClient := jsonclient.DefaultClient
	jsonclient.DefaultClient = jsonclient.NewWithHTTPClient(client)
	defer func() {
		jsonclient.DefaultClient = previousClient
	}()

	return provider.Send(message, "{{data}}")
}

func telegramHTTPClient(settings TelegramSettings) (*http.Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = (&net.Dialer{
		Timeout:   telegramSendTimeout,
		KeepAlive: 30 * time.Second,
	}).DialContext
	transport.TLSHandshakeTimeout = 5 * time.Second
	transport.ResponseHeaderTimeout = telegramSendTimeout
	transport.ExpectContinueTimeout = time.Second
	if settings.ProxyEnabled {
		proxyURL, err := telegramProxyURL(settings)
		if err != nil {
			return nil, err
		}
		transport.Proxy = http.ProxyURL(proxyURL)
	}
	return &http.Client{
		Transport: telegramErrorTransport{base: transport},
		Timeout:   telegramSendTimeout,
	}, nil
}

type telegramErrorTransport struct {
	base http.RoundTripper
}

func (t telegramErrorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err == nil {
		return resp, nil
	}

	body := fmt.Sprintf(`{"ok":false,"description":%q}`, telegramTransportError(err))
	return &http.Response{
		StatusCode: http.StatusBadGateway,
		Status:     "502 Bad Gateway",
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}, nil
}

func telegramTransportError(err error) string {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return "telegram request timed out after 10 seconds"
	}
	message := strings.ToLower(err.Error())
	if strings.Contains(message, "proxy") || strings.Contains(message, "socks") {
		return "telegram request failed through SOCKS5 proxy"
	}
	if strings.Contains(message, "connection refused") || strings.Contains(message, "connect:") {
		return "telegram network connection failed"
	}
	return "telegram request failed"
}

func telegramProxyURL(settings TelegramSettings) (*url.URL, error) {
	if settings.ProxyHost == "" || settings.ProxyPort < 1 || settings.ProxyPort > 65535 {
		return nil, fmt.Errorf("invalid SOCKS5 proxy settings")
	}
	proxyURL := &url.URL{
		Scheme: "socks5",
		Host:   net.JoinHostPort(settings.ProxyHost, strconv.Itoa(settings.ProxyPort)),
	}
	if settings.ProxyUsername != "" || settings.ProxyPassword != "" {
		proxyURL.User = url.UserPassword(settings.ProxyUsername, settings.ProxyPassword)
	}
	return proxyURL, nil
}

func buildTelegramRequestMessages(webhook Webhook, request CapturedRequest, host string) []string {
	rawRequest := sanitizeTelegramCodeBlock(formatHTTPNotificationRequest(request, host))
	chunks := splitMessageRunes(rawRequest, telegramMessageLimit)
	if len(chunks) == 0 {
		chunks = []string{""}
	}

	meta := fmt.Sprintf(
		"IP: %s\nRequest time: %s\nWebhook: %s",
		request.RemoteIP,
		formatTelegramTime(request.CreatedAt),
		webhook.Slug,
	)

	messages := make([]string, 0, len(chunks))
	for i, chunk := range chunks {
		label := ""
		if len(chunks) > 1 {
			label = fmt.Sprintf("\nPart %d/%d", i+1, len(chunks))
		}
		prefix := ""
		if i == 0 {
			prefix = meta + label + "\n\n"
		} else {
			prefix = fmt.Sprintf("Webhook: %s%s\n\n", webhook.Slug, label)
		}
		messages = append(messages, prefix+"```http\n"+chunk+"\n```")
	}
	return messages
}

func formatHTTPNotificationRequest(request CapturedRequest, host string) string {
	var headers map[string][]string
	_ = json.Unmarshal(request.Headers, &headers)

	target := request.Path
	if request.QueryString != "" {
		target += "?" + request.QueryString
	}

	var out bytes.Buffer
	fmt.Fprintf(&out, "%s %s HTTP/1.1\n", request.Method, target)
	if host != "" {
		fmt.Fprintf(&out, "Host: %s\n", host)
	}

	names := make([]string, 0, len(headers))
	for name := range headers {
		if strings.EqualFold(name, "Host") {
			continue
		}
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return strings.ToLower(names[i]) < strings.ToLower(names[j])
	})
	for _, name := range names {
		for _, value := range headers[name] {
			fmt.Fprintf(&out, "%s: %s\n", name, value)
		}
	}

	if len(request.Body) == 0 && !request.BodyTruncated {
		return strings.TrimRight(out.String(), "\n")
	}
	out.WriteByte('\n')
	out.WriteString(notificationBody(request.Body))
	if request.BodyTruncated {
		out.WriteString("\n\n[body truncated]")
	}
	return out.String()
}

func notificationBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	if utf8.Valid(body) {
		return string(body)
	}
	return "[binary body omitted]"
}

func sanitizeTelegramCodeBlock(value string) string {
	return strings.ReplaceAll(value, "```", "``\u200b`")
}

func splitMessageRunes(value string, limit int) []string {
	runes := []rune(value)
	if len(runes) <= limit {
		return []string{value}
	}
	var chunks []string
	for len(runes) > 0 {
		size := limit
		if len(runes) < size {
			size = len(runes)
		}
		chunks = append(chunks, string(runes[:size]))
		runes = runes[size:]
	}
	return chunks
}

func formatTelegramTime(value time.Time) string {
	return value.Local().Format("2 January 2006 15:04:05")
}

func logNotificationError(context string, err error) {
	if err != nil {
		log.Printf("telegram notification: %s: %v", context, err)
	}
}
