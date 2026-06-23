export function displayWebhookName(webhook) {
  return webhook?.name || webhook?.slug || ''
}

export function selectorWebhookMeta(webhook) {
  const count = webhook?.requestCount || 0
  const countLabel = `${count} ${count === 1 ? 'request' : 'requests'}`
  if (!webhook?.name || webhook.name === webhook.slug) return countLabel
  return `${webhook.slug} · ${countLabel}`
}

export function symbolCount(value) {
  return Array.from(value || '').length
}

export function clamp(value, min, max) {
  return Math.min(Math.max(value, min), max)
}

export function formatTime(value) {
  return new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(new Date(value))
}

export function formatRequestPath(request, slug = '') {
  const target = request?.target || ''
  if (!slug || !target.startsWith(`/at/${slug}`)) return target || '/'
  const rest = target.slice(`/at/${slug}`.length)
  return rest === '' ? '/' : rest
}

export function formatSearchResultTarget(result) {
  const target = result?.request?.target || result?.request?.path || ''
  const slug = result?.webhookSlug || ''
  if (!slug || !target.startsWith(`/at/${slug}`)) return target || '/'
  const rest = target.slice(`/at/${slug}`.length)
  return rest === '' ? '/' : rest
}

export function formatBreakableUri(value) {
  return (value || '').replace(/([/?&])/g, '$1\u200b')
}

export function formatReadableDate(value) {
  const parts = new Intl.DateTimeFormat(undefined, {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
    hour12: false
  }).formatToParts(new Date(value))
  const byType = Object.fromEntries(parts.map((part) => [part.type, part.value]))
  return `${byType.day} ${byType.month} ${byType.year} ${byType.hour}:${byType.minute}:${byType.second}`
}

export function formatBytes(value) {
  if (value < 0) return 'unknown'
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KiB`
  return `${(value / 1024 / 1024).toFixed(1)} MiB`
}
