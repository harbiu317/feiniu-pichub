import { defineStore } from 'pinia'
import { api, clearToken, getToken, setToken } from '@/api'
import { ref, computed } from 'vue'

export const useUserStore = defineStore('user', () => {
  const user = ref<any>(null)
  const loading = ref(false)
  const isLoggedIn = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.role === 'admin')

  async function login(username: string, password: string) {
    const res = await api.login(username, password)
    setToken(res.token)
    user.value = res.user
  }

  async function fetchMe() {
    if (!getToken()) return
    loading.value = true
    try {
      user.value = await api.me()
    } catch {
      clearToken()
      user.value = null
    } finally {
      loading.value = false
    }
  }

  function logout() {
    clearToken()
    user.value = null
  }

  return { user, loading, isLoggedIn, isAdmin, login, fetchMe, logout }
})
