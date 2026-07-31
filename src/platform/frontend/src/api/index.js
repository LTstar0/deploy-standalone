import axios from 'axios'

const api = axios.create({
  baseURL: '/api',
  timeout: 60000
})

// Request interceptor to attach token
api.interceptors.request.use(config => {
  const token = localStorage.getItem('deploy_token')
  if (token) {
    config.headers['X-Deploy-Token'] = token
  }
  return config
}, error => {
  return Promise.reject(error)
})

// ── Authentication ───
export const verifyToken = (token) => {
  return api.get('/verify', {
    headers: { 'X-Deploy-Token': token }
  })
}

// ── Packages ───
export const uploadPackage = (file, onProgress) => {
  const form = new FormData()
  form.append('file', file)
  return api.post('/packages/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    onUploadProgress: e => {
      if (onProgress && e.total) onProgress(Math.round(e.loaded / e.total * 100))
    }
  })
}

export const getPackages = () => api.get('/packages')
export const getPackage = (id) => api.get(`/packages/${id}`)
export const deletePackage = (id) => api.delete(`/packages/${id}`)

// ── Deploy ───
export const startDeploy = (packageId, selectedApps) =>
  api.post('/deploy/start', { package_id: packageId, selected_apps: selectedApps })

export const getTask = (id) => api.get(`/deploy/tasks/${id}`)
export const getHistory = () => api.get('/deploy/history')
export const getHistoryDetail = (id) => api.get(`/deploy/history/${id}`)
export const rollback = (historyId) => api.post(`/deploy/rollback/${historyId}`)

// ── WebSocket ───
export const connectWS = (taskId, onMessage, onClose) => {
  const proto = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const token = localStorage.getItem('deploy_token') || ''
  const tokenParam = token ? `?token=${encodeURIComponent(token)}` : ''
  const ws = new WebSocket(`${proto}//${location.host}/ws/deploy/${taskId}${tokenParam}`)
  ws.onmessage = (e) => {
    try {
      onMessage(JSON.parse(e.data))
    } catch {}
  }
  ws.onclose = () => onClose && onClose()
  ws.onerror = () => onClose && onClose()
  return ws
}

