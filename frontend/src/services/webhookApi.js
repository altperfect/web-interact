import { api, csrfHeaders } from './api'

export function getSession() {
  return api('/api/session')
}

export function listWebhooks() {
  return api('/api/webhooks')
}

export function createWebhook(session) {
  return api('/api/webhooks', {
    method: 'POST',
    headers: csrfHeaders(session)
  })
}

export function deleteWebhook(slug, session) {
  return api(`/api/webhooks/${slug}`, {
    method: 'DELETE',
    headers: csrfHeaders(session)
  })
}

export function setShareEnabled(slug, enabled, session) {
  return api(`/api/webhooks/${slug}/share`, {
    method: 'PATCH',
    headers: csrfHeaders(session),
    body: JSON.stringify({ enabled })
  })
}

export function setWebhookTelegramEnabled(slug, enabled, session) {
  return api(`/api/webhooks/${slug}/telegram`, {
    method: 'PATCH',
    headers: csrfHeaders(session),
    body: JSON.stringify({ enabled })
  })
}

export function setWebhookName(slug, name, session) {
  return api(`/api/webhooks/${slug}/name`, {
    method: 'PATCH',
    headers: csrfHeaders(session),
    body: JSON.stringify({ name })
  })
}

export function saveWebhookResponse(slug, payload, session) {
  return api(`/api/webhooks/${slug}/response`, {
    method: 'PATCH',
    headers: csrfHeaders(session),
    body: JSON.stringify(payload)
  })
}

export function getSharedWebhook(slug, token) {
  return api(`/api/share/${slug}?id=${encodeURIComponent(token)}`)
}

export function listRequests(slug, { shareMode = false, shareToken = '' } = {}) {
  const path = shareMode
    ? `/api/share/${slug}/requests?id=${encodeURIComponent(shareToken)}`
    : `/api/webhooks/${slug}/requests`
  return api(path)
}

export function getRequest(slug, requestId, { shareMode = false, shareToken = '' } = {}) {
  const path = shareMode
    ? `/api/share/${slug}/requests/${requestId}?id=${encodeURIComponent(shareToken)}`
    : `/api/webhooks/${slug}/requests/${requestId}`
  return api(path)
}

export function deleteRequest(slug, requestId, session) {
  return api(`/api/webhooks/${slug}/requests/${requestId}`, {
    method: 'DELETE',
    headers: csrfHeaders(session)
  })
}

export function saveRequestNote(slug, requestId, note, session) {
  return api(`/api/webhooks/${slug}/requests/${requestId}/note`, {
    method: 'PATCH',
    headers: csrfHeaders(session),
    body: JSON.stringify({ note })
  })
}

export function searchRequests({ query, webhook = '', limit = 40 }, signal) {
  const params = new URLSearchParams({
    q: query,
    limit: String(limit)
  })
  if (webhook) params.set('webhook', webhook)
  return api(`/api/search?${params.toString()}`, { signal })
}

export function getTelegramSettings() {
  return api('/api/telegram')
}

export function saveTelegramSettings(payload, session) {
  return api('/api/telegram', {
    method: 'PATCH',
    headers: csrfHeaders(session),
    body: JSON.stringify(payload)
  })
}

export function testTelegramSettings(payload, session, signal) {
  return api('/api/telegram/test', {
    method: 'POST',
    headers: csrfHeaders(session),
    body: JSON.stringify(payload),
    signal
  })
}
