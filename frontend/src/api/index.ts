const TOKEN_KEY = 'pichub_token'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(t: string) {
  localStorage.setItem(TOKEN_KEY, t)
}

export function clearToken() {
  localStorage.removeItem(TOKEN_KEY)
}

async function request<T = any>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (!headers.has('Content-Type') && !(init.body instanceof FormData)) {
    headers.set('Content-Type', 'application/json')
  }
  const token = getToken()
  if (token) headers.set('Authorization', `Bearer ${token}`)
  const res = await fetch(path, { ...init, headers })
  const ct = res.headers.get('content-type') || ''
  const data = ct.includes('application/json') ? await res.json() : await res.text()
  if (!res.ok) {
    const msg = (data && typeof data === 'object' && (data as any).error) || res.statusText
    throw new Error(msg)
  }
  return data as T
}

export const api = {
  // meta
  meta: () => request<{ version: string; storage_driver: string }>('/api/meta'),

  // setup
  setupStatus: () => request<{ needs_setup: boolean }>('/api/setup/status'),
  setupInit: (body: { username: string; password: string; allow_anonymous?: boolean }) =>
    request<{ token: string; user: any }>('/api/setup/init', {
      method: 'POST',
      body: JSON.stringify(body),
    }),

  // auth
  login: (username: string, password: string) =>
    request<{ token: string; user: any }>('/api/auth/login', {
      method: 'POST',
      body: JSON.stringify({ username, password }),
    }),
  me: () => request<any>('/api/auth/me'),

  // stats
  stats: () => request<any>('/api/stats'),

  // upload
  upload: (files: File[], onProgress?: (p: number) => void) => {
    const form = new FormData()
    files.forEach(f => form.append('files', f))
    return new Promise<any>((resolve, reject) => {
      const xhr = new XMLHttpRequest()
      xhr.open('POST', '/api/upload')
      const token = getToken()
      if (token) xhr.setRequestHeader('Authorization', `Bearer ${token}`)
      xhr.upload.onprogress = e => {
        if (e.lengthComputable && onProgress) onProgress(Math.round((e.loaded / e.total) * 100))
      }
      xhr.onload = () => {
        try {
          const data = JSON.parse(xhr.responseText)
          if (xhr.status >= 200 && xhr.status < 300) resolve(data)
          else reject(new Error(data.error || xhr.statusText))
        } catch {
          reject(new Error(xhr.statusText))
        }
      }
      xhr.onerror = () => reject(new Error('网络错误'))
      xhr.send(form)
    })
  },

  // images
  listImages: (params: { page?: number; size?: number; q?: string; album?: number } = {}) => {
    const q = new URLSearchParams()
    Object.entries(params).forEach(([k, v]) => v != null && q.set(k, String(v)))
    return request<{ items: any[]; total: number; page: number; size: number }>(`/api/images?${q}`)
  },
  deleteImage: (id: number) => request(`/api/images/${id}`, { method: 'DELETE' }),
  batchDelete: (ids: number[]) =>
    request('/api/images/batch-delete', { method: 'POST', body: JSON.stringify({ ids }) }),
  moveImages: (imageIds: number[], albumId: number) =>
    request('/api/images/move', {
      method: 'POST',
      body: JSON.stringify({ image_ids: imageIds, album_id: albumId }),
    }),

  // albums
  listAlbums: () => request<any[]>('/api/albums'),
  createAlbum: (name: string, description = '', isPublic = false) =>
    request('/api/albums', {
      method: 'POST',
      body: JSON.stringify({ name, description, is_public: isPublic }),
    }),
  updateAlbum: (id: number, body: any) =>
    request(`/api/albums/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteAlbum: (id: number) => request(`/api/albums/${id}`, { method: 'DELETE' }),

  // tokens
  listTokens: () => request<any[]>('/api/tokens'),
  createToken: (name: string, days = 0) =>
    request('/api/tokens', { method: 'POST', body: JSON.stringify({ name, days }) }),
  deleteToken: (id: number) => request(`/api/tokens/${id}`, { method: 'DELETE' }),

  // admin
  admin: {
    listUsers: () => request<any[]>('/api/admin/users'),
    createUser: (body: any) =>
      request('/api/admin/users', { method: 'POST', body: JSON.stringify(body) }),
    updateUser: (id: number, body: any) =>
      request(`/api/admin/users/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
    deleteUser: (id: number) => request(`/api/admin/users/${id}`, { method: 'DELETE' }),
    getSettings: () => request<any>('/api/admin/settings'),
    updateSettings: (body: any) =>
      request('/api/admin/settings', { method: 'PUT', body: JSON.stringify(body) }),
    testStorage: (body: any) =>
      request<{ ok: boolean; message: string }>('/api/admin/storage/test', {
        method: 'POST',
        body: JSON.stringify(body),
      }),
  },
}
