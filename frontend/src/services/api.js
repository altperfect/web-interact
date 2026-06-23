export async function api(path, options = {}) {
  const response = await fetch(path, {
    credentials: 'same-origin',
    ...options,
    headers: {
      Accept: 'application/json',
      ...(options.headers || {})
    }
  })
  if (response.status === 204) return null
  const contentType = response.headers.get('content-type') || ''
  const data = contentType.includes('application/json') ? await response.json() : {}
  if (!response.ok) {
    throw new Error(data.error || `Request failed with ${response.status}`)
  }
  return data
}

export function csrfHeaders(session) {
  return {
    'Content-Type': 'application/json',
    'X-CSRF-Token': session?.csrfToken || ''
  }
}
