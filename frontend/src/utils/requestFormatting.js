export function formatHTTPClipboardRequest(request) {
  const target = request.target || request.path || '/'
  const lines = [`${request.method || 'GET'} ${target} HTTP/1.1`]
  const host = requestHost(request)
  if (host) lines.push(`Host: ${host}`)

  const headers = Object.entries(request.headers || {})
    .filter(([name]) => !nameEquals(name, 'Host'))
    .sort(([a], [b]) => a.toLowerCase().localeCompare(b.toLowerCase()))

  for (const [name, values] of headers) {
    for (const value of values || []) {
      lines.push(`${name}: ${value}`)
    }
  }

  if (!request.bodySize && !request.bodyTruncated) {
    return lines.join('\n')
  }

  const body = request.bodyEncoding === 'base64'
    ? '[binary body omitted]'
    : (request.bodyText || '')
  const suffix = request.bodyTruncated ? `${body ? '\n\n' : ''}[body truncated]` : ''
  return `${lines.join('\n')}\n\n${body}${suffix}`
}

export function requestHost(request) {
  try {
    return new URL(request.detailUrl || window.location.href, window.location.origin).host
  } catch {
    return window.location.host
  }
}

export function nameEquals(left, right) {
  return String(left).toLowerCase() === String(right).toLowerCase()
}

export function headerNameCopyTarget(name) {
  return `header-name:${name}`
}

export function headerValueCopyTarget(name) {
  return `header-value:${name}`
}

export function headerValueText(values) {
  return values.join(', ')
}
