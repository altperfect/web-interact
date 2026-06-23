export function loadStoredPrefs(key) {
  try {
    return JSON.parse(window.localStorage.getItem(key) || '{}')
  } catch {
    return {}
  }
}

export function loadStoredValue(key) {
  try {
    return window.localStorage.getItem(key) || ''
  } catch {
    return ''
  }
}

export function saveStoredValue(key, value) {
  try {
    if (value) {
      window.localStorage.setItem(key, value)
    } else {
      window.localStorage.removeItem(key)
    }
  } catch {
    // Ignore storage failures; route-based navigation still works.
  }
}

export function saveStoredPrefs(key, value) {
  window.localStorage.setItem(key, JSON.stringify(value))
}
