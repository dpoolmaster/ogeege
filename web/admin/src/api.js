function cookie(name) {
  const prefix = name + '='
  const item = document.cookie.split('; ').find((value) => value.startsWith(prefix))
  return item ? decodeURIComponent(item.slice(prefix.length)) : ''
}

export async function api(path, options = {}) {
  const method = options.method || 'GET'
  const headers = { ...(options.headers || {}) }
  if (options.body && !(options.body instanceof FormData)) {
    headers['Content-Type'] = 'application/json'
  }
  if (method !== 'GET' && method !== 'HEAD') {
    headers['X-CSRF-Token'] = cookie('csrf_token')
  }

  const response = await fetch('/api' + path, {
    credentials: 'same-origin',
    ...options,
    method,
    headers
  })

  if (response.status === 401) {
    window.location.href = '/login'
    throw new Error('Требуется вход')
  }
  if (response.status === 204) {
    return null
  }

  const data = await response.json().catch(() => ({}))
  if (!response.ok) {
    const error = data.error || {}
    const failure = new Error(error.message || 'Ошибка запроса')
    failure.status = response.status
    throw failure
  }

  return data
}

export function jsonBody(value) {
  return JSON.stringify(value)
}
