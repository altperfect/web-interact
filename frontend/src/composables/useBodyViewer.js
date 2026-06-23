import { computed, ref } from 'vue'
import { nameEquals } from '../utils/requestFormatting'

export function useBodyViewer(selectedRequest) {
  const enhancedBodyView = ref(true)

  const headerRows = computed(() => {
    if (!selectedRequest.value?.headers) return []
    return Object.entries(selectedRequest.value.headers).sort(([a], [b]) => a.localeCompare(b))
  })

  const bodyDisplay = computed(() => {
    if (!selectedRequest.value) return ''
    if (selectedRequest.value.bodyEncoding === 'base64') {
      return selectedRequest.value.bodyBase64 || 'No body'
    }
    const suffix = selectedRequest.value.bodyTruncated ? '\n\n[body truncated]' : ''
    return (selectedRequest.value.bodyText || 'No body') + suffix
  })

  const hasBody = computed(() => {
    if (!selectedRequest.value) return false
    return selectedRequest.value.bodySize > 0 || selectedRequest.value.bodyText !== '' || selectedRequest.value.bodyBase64 !== ''
  })

  const defaultBodyView = () => ({
    kind: 'default',
    label: '',
    text: bodyDisplay.value
  })

  const detectedBodyView = computed(() => detectBodyView(selectedRequest.value, hasBody.value, defaultBodyView))

  const bodyHasEnhancedView = computed(() => detectedBodyView.value.kind !== 'default')

  const bodyView = computed(() => {
    if (enhancedBodyView.value && bodyHasEnhancedView.value) return detectedBodyView.value
    return defaultBodyView()
  })

  function toggleBodyView() {
    enhancedBodyView.value = !enhancedBodyView.value
  }

  return {
    enhancedBodyView,
    headerRows,
    bodyDisplay,
    hasBody,
    detectedBodyView,
    bodyHasEnhancedView,
    bodyView,
    toggleBodyView
  }
}

function detectBodyView(request, hasBody, defaultBodyView) {
  if (!request || !hasBody) return defaultBodyView()
  const contentType = requestContentType(request)
  const normalizedType = contentType.toLowerCase().split(';')[0].trim()

  if (contentType.toLowerCase().startsWith('multipart/')) {
    const multipart = parseMultipartBody(request, contentType)
    if (multipart) return multipart
  }

  const imageType = normalizedType.startsWith('image/') ? normalizedType : sniffImageType(request.bodyBase64 || '')
  if (imageType && request.bodyBase64) {
    return {
      kind: 'image',
      label: 'Image',
      src: `data:${imageType};base64,${request.bodyBase64}`,
      alt: `${imageType} request body`
    }
  }

  if (request.bodyEncoding !== 'text') return defaultBodyView()
  const text = request.bodyText || ''
  const trimmed = text.trim()

  if (isJSONContent(contentType, trimmed)) {
    const pretty = prettyJSON(text)
    if (pretty) return { kind: 'json', label: 'JSON', text: pretty }
  }

  if (isHTMLContent(contentType, trimmed)) {
    const pretty = prettyMarkup(text, 'html')
    if (pretty) return { kind: 'html', label: 'HTML', text: pretty }
  }

  if (isXMLContent(contentType, trimmed)) {
    const pretty = prettyMarkup(text, 'xml')
    if (pretty) return { kind: 'xml', label: 'XML', text: pretty }
  }

  return defaultBodyView()
}

function requestContentType(request) {
  const entry = Object.entries(request?.headers || {}).find(([name]) => nameEquals(name, 'Content-Type'))
  return entry?.[1]?.[0] || ''
}

function isJSONContent(contentType, trimmed) {
  const type = contentType.toLowerCase()
  return type.includes('/json') || type.includes('+json') || trimmed.startsWith('{') || trimmed.startsWith('[')
}

function isHTMLContent(contentType, trimmed) {
  const type = contentType.toLowerCase()
  return type.includes('text/html') || /^<!doctype\s+html/i.test(trimmed) || /^<html[\s>]/i.test(trimmed)
}

function isXMLContent(contentType, trimmed) {
  const type = contentType.toLowerCase()
  return type.includes('/xml') || type.includes('+xml') || /^<\?xml[\s>]/i.test(trimmed)
}

function sniffImageType(base64) {
  if (!base64) return ''
  if (base64.startsWith('/9j/')) return 'image/jpeg'
  if (base64.startsWith('iVBORw0KGgo')) return 'image/png'
  if (base64.startsWith('R0lGOD')) return 'image/gif'
  try {
    const prefix = window.atob(base64.slice(0, 24))
    if (prefix.startsWith('RIFF') && prefix.slice(8, 12) === 'WEBP') return 'image/webp'
  } catch {
    return ''
  }
  return ''
}

function prettyJSON(value) {
  try {
    return JSON.stringify(JSON.parse(value), null, 2)
  } catch {
    return ''
  }
}

function prettyMarkup(value, mode) {
  const parsed = parseMarkup(value, mode)
  if (!parsed) return ''
  return formatMarkup(parsed)
}

function parseMarkup(value, mode) {
  try {
    const parser = new DOMParser()
    const document = parser.parseFromString(value, mode === 'html' ? 'text/html' : 'application/xml')
    if (mode === 'xml' && document.querySelector('parsererror')) return ''
    const trimmed = value.trim()
    const serialized = mode === 'html'
      ? (/^<!doctype\s+html/i.test(trimmed) || /^<html[\s>]/i.test(trimmed) ? document.documentElement.outerHTML : document.body.innerHTML)
      : new XMLSerializer().serializeToString(document)
    return serialized || value
  } catch {
    return ''
  }
}

function formatMarkup(value) {
  const compact = value
    .replace(/>\s+</g, '><')
    .replace(/></g, '>\n<')
  const lines = compact.split('\n')
  let depth = 0
  return lines.map((line) => {
    const trimmed = line.trim()
    if (/^<\//.test(trimmed)) depth = Math.max(depth - 1, 0)
    const formatted = `${'  '.repeat(depth)}${trimmed}`
    if (
      /^<[^!?/][^>]*[^/]?>$/.test(trimmed) &&
      !/^<[^>]+>.*<\/[^>]+>$/.test(trimmed) &&
      !isVoidElement(trimmed)
    ) {
      depth += 1
    }
    return formatted
  }).join('\n')
}

function isVoidElement(tag) {
  return /^<(area|base|br|col|embed|hr|img|input|link|meta|param|source|track|wbr)(\s|>)/i.test(tag)
}

function parseMultipartBody(request, contentType) {
  const boundary = multipartBoundary(contentType)
  if (!boundary) return null
  const raw = requestBodyRaw(request)
  if (!raw.value) return null
  const delimiter = `--${boundary}`
  const chunks = raw.value.split(delimiter).slice(1)
  const parts = chunks
    .map((chunk) => parseMultipartPart(chunk, raw.binary))
    .filter(Boolean)
  if (parts.length === 0) return null
  return {
    kind: 'multipart',
    label: 'Multipart',
    parts
  }
}

function multipartBoundary(contentType) {
  const match = contentType.match(/boundary=(?:"([^"]+)"|([^;]+))/i)
  return (match?.[1] || match?.[2] || '').trim()
}

function requestBodyRaw(request) {
  if (request.bodyEncoding === 'base64') {
    try {
      return { value: window.atob(request.bodyBase64 || ''), binary: true }
    } catch {
      return { value: '', binary: true }
    }
  }
  return { value: request.bodyText || '', binary: false }
}

function parseMultipartPart(chunk, sourceIsBinary) {
  let value = chunk
  if (value.startsWith('--')) return null
  value = value.replace(/^\r?\n/, '').replace(/\r?\n--$/, '').replace(/\r?\n$/, '')
  if (!value.trim()) return null
  const separator = value.indexOf('\r\n\r\n') === -1 ? '\n\n' : '\r\n\r\n'
  const separatorIndex = value.indexOf(separator)
  if (separatorIndex === -1) return null
  const headerText = value.slice(0, separatorIndex)
  const body = value.slice(separatorIndex + separator.length)
  const headers = parsePartHeaders(headerText)
  const contentType = headerValue(headers, 'Content-Type')
  const disposition = parseContentDisposition(headerValue(headers, 'Content-Disposition'))
  const title = disposition.filename || disposition.name || contentType || 'Part'
  const normalizedType = contentType.toLowerCase().split(';')[0].trim()
  const imageSrc = normalizedType.startsWith('image/')
    ? `data:${normalizedType};base64,${binaryStringToBase64(body)}`
    : ''
  const text = imageSrc ? '' : multipartPartText(body, sourceIsBinary, contentType)
  return {
    title,
    headers,
    body: text || 'No body',
    imageSrc,
    size: body.length
  }
}

function parsePartHeaders(value) {
  return value.split(/\r?\n/)
    .map((line) => {
      const index = line.indexOf(':')
      if (index === -1) return null
      return [line.slice(0, index).trim(), line.slice(index + 1).trim()]
    })
    .filter(Boolean)
}

function headerValue(headers, name) {
  const match = headers.find(([headerName]) => nameEquals(headerName, name))
  return match?.[1] || ''
}

function parseContentDisposition(value) {
  const out = {}
  value.split(';').slice(1).forEach((part) => {
    const [name, ...rest] = part.split('=')
    if (!name || rest.length === 0) return
    out[name.trim().toLowerCase()] = rest.join('=').trim().replace(/^"|"$/g, '')
  })
  return out
}

function multipartPartText(value, sourceIsBinary, contentType) {
  const text = sourceIsBinary ? decodeBinaryString(value) : value
  if (isJSONContent(contentType, text.trim())) return prettyJSON(text) || text
  if (isHTMLContent(contentType, text.trim())) return prettyMarkup(text, 'html') || text
  if (isXMLContent(contentType, text.trim())) return prettyMarkup(text, 'xml') || text
  return text
}

function decodeBinaryString(value) {
  try {
    const bytes = Uint8Array.from(value, (char) => char.charCodeAt(0) & 0xff)
    return new TextDecoder().decode(bytes)
  } catch {
    return value
  }
}

function binaryStringToBase64(value) {
  try {
    return window.btoa(value)
  } catch {
    return ''
  }
}
