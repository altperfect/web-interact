export function parseRoute() {
  const parts = window.location.pathname.split('/').filter(Boolean)
  if (parts[0] === 'share') {
    return {
      mode: 'share',
      slug: parts[1] || '',
      requestId: parts[2] || '',
      shareToken: new URLSearchParams(window.location.search).get('id') || ''
    }
  }
  if (parts[0] === 'at' && parts.length >= 3) {
    return {
      mode: 'owner',
      slug: parts[1],
      requestId: parts[2],
      shareToken: ''
    }
  }
  return { mode: 'owner', slug: '', requestId: '', shareToken: '' }
}

export function ownerRootPath() {
  return '/'
}

export function shareWebhookPath(slug, token) {
  return `/share/${slug}?id=${encodeURIComponent(token)}`
}

export function requestDetailPath({ shareMode, slug, requestId, shareToken }) {
  return shareMode
    ? `/share/${slug}/${requestId}?id=${encodeURIComponent(shareToken)}`
    : `/at/${slug}/${requestId}`
}
