const API_BASE = '/api'
let authToken = localStorage.getItem('lightboot_token') || ''

export function setToken(token) { authToken = token; localStorage.setItem('lightboot_token', token) }
export function getToken() { return authToken }
export function clearToken() { authToken = ''; localStorage.removeItem('lightboot_token') }

async function request(path, options = {}) {
  const headers = { ...options.headers }
  if (authToken) headers['Authorization'] = 'Bearer ' + authToken
  const resp = await fetch(API_BASE + path, { ...options, headers })
  if (resp.status === 401) { clearToken(); window.location.hash = '#/login'; throw new Error('Unauthorized') }
  if (!resp.ok) { const msg = await resp.text().catch(() => resp.statusText); throw new Error(msg || 'Request failed') }
  const ct = resp.headers.get('Content-Type') || ''
  if (ct.includes('application/json')) return resp.json()
  return resp.text()
}

export function apiGet(path) { return request(path) }
export function apiPost(path, body) {
  return request(path, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) })
}
export function apiDelete(path) { return request(path, { method: 'DELETE' }) }

export function apiUpload(path, file, onProgress) {
  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest()
    xhr.open('POST', API_BASE + path)
    if (authToken) xhr.setRequestHeader('Authorization', 'Bearer ' + authToken)
    xhr.upload.onprogress = (e) => { if (e.lengthComputable && onProgress) onProgress(Math.round((e.loaded / e.total) * 100)) }
    xhr.onload = () => {
      if (xhr.status === 401) { clearToken(); window.location.hash = '#/login'; reject(new Error('Unauthorized')); return }
      if (xhr.status >= 400) { reject(new Error(xhr.responseText || 'Upload failed')); return }
      try { resolve(JSON.parse(xhr.responseText)) } catch { resolve(xhr.responseText) }
    }
    xhr.onerror = () => reject(new Error('Network error'))
    const fd = new FormData()
    fd.append('file', file)
    xhr.send(fd)
  })
}

export function createLogStream(onMessage, onError) {
  const es = new EventSource(API_BASE + '/logs/stream')
  es.onmessage = (e) => { try { onMessage(JSON.parse(e.data)) } catch { onMessage({ message: e.data }) } }
  es.onerror = () => { if (onError) onError(); es.close() }
  return es
}
