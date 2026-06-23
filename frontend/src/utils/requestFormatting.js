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

export function formatHTTPResponse(request) {
  const statusCode = Number(request?.responseStatusCode) || 200
  const reason = responseReasonPhrase(statusCode)
  const lines = [`HTTP/1.1 ${statusCode}${reason ? ` ${reason}` : ''}`]
  const headers = normalizeResponseHeaders(request?.responseHeaders)
  const body = request?.responseBody || ''

  for (const header of headers) {
    lines.push(`${header.name}: ${header.value}`)
  }

  if (!headers.some((header) => nameEquals(header.name, 'Content-Length'))) {
    lines.push(`Content-Length: ${byteLength(body)}`)
  }

  return `${lines.join('\n')}\n\n${body}`
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

function normalizeResponseHeaders(headers) {
  if (!Array.isArray(headers)) return []
  return headers
    .map((header) => ({
      name: String(header?.name || '').trim(),
      value: String(header?.value || '')
    }))
    .filter((header) => header.name)
}

function byteLength(value) {
  if (typeof TextEncoder !== 'undefined') {
    return new TextEncoder().encode(value).length
  }
  return String(value).length
}

function responseReasonPhrase(statusCode) {
  return responseReasons[statusCode] || ''
}

const responseReasons = {
  100: 'Continue',
  101: 'Switching Protocols',
  102: 'Processing',
  103: 'Early Hints',
  200: 'OK',
  201: 'Created',
  202: 'Accepted',
  203: 'Non-Authoritative Information',
  204: 'No Content',
  205: 'Reset Content',
  206: 'Partial Content',
  207: 'Multi-Status',
  208: 'Already Reported',
  226: 'IM Used',
  300: 'Multiple Choices',
  301: 'Moved Permanently',
  302: 'Found',
  303: 'See Other',
  304: 'Not Modified',
  307: 'Temporary Redirect',
  308: 'Permanent Redirect',
  400: 'Bad Request',
  401: 'Unauthorized',
  402: 'Payment Required',
  403: 'Forbidden',
  404: 'Not Found',
  405: 'Method Not Allowed',
  406: 'Not Acceptable',
  407: 'Proxy Authentication Required',
  408: 'Request Timeout',
  409: 'Conflict',
  410: 'Gone',
  411: 'Length Required',
  412: 'Precondition Failed',
  413: 'Content Too Large',
  414: 'URI Too Long',
  415: 'Unsupported Media Type',
  416: 'Range Not Satisfiable',
  417: 'Expectation Failed',
  418: "I'm a Teapot",
  421: 'Misdirected Request',
  422: 'Unprocessable Content',
  423: 'Locked',
  424: 'Failed Dependency',
  425: 'Too Early',
  426: 'Upgrade Required',
  428: 'Precondition Required',
  429: 'Too Many Requests',
  431: 'Request Header Fields Too Large',
  451: 'Unavailable For Legal Reasons',
  500: 'Internal Server Error',
  501: 'Not Implemented',
  502: 'Bad Gateway',
  503: 'Service Unavailable',
  504: 'Gateway Timeout',
  505: 'HTTP Version Not Supported',
  506: 'Variant Also Negotiates',
  507: 'Insufficient Storage',
  508: 'Loop Detected',
  510: 'Not Extended',
  511: 'Network Authentication Required'
}
