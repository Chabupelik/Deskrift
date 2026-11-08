import { defineStore } from 'pinia'
import { ref } from 'vue'
import { authApi } from '../api/client'

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('token'))
  const isAuth = ref(!!token.value)

  async function loginWithTelegram(initData: string) {
    const { data } = await authApi.telegram(initData)
    token.value = data.access_token
    localStorage.setItem('token', data.access_token)
    isAuth.value = true
  }

  function logout() {
    token.value = null
    isAuth.value = false
    localStorage.removeItem('token')
  }

  return { token, isAuth, loginWithTelegram, logout }
})

