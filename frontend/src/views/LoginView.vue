<template>
  <div class="login-page">
    <div class="login-card">
      <div class="header">
        <div class="brand-icon">
          <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect width="20" height="14" x="2" y="3" rx="2" />
            <line x1="8" x2="16" y1="21" y2="21" />
            <line x1="12" x2="12" y1="17" y2="21" />
          </svg>
        </div>
        <h1 class="brand-name">Deskrift</h1>
        <p class="brand-desc">Низколатентный доступ к рабочему столу</p>
      </div>

      <div v-if="state === 'idle'" class="content">
        <button class="btn-primary" @click="startLogin">
          <svg class="tg-icon" width="18" height="18" viewBox="0 0 24 24" fill="currentColor">
            <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4.64 6.8c-.15 1.58-.8 5.42-1.13 7.19-.14.75-.42 1-.68 1.03-.58.05-1.02-.38-1.58-.75-.88-.58-1.38-.94-2.23-1.5-.99-.65-.35-1.01.22-1.59.15-.15 2.71-2.48 2.76-2.69a.2.2 0 00-.05-.18c-.06-.05-.14-.03-.21-.02-.09.02-1.49.95-4.22 2.79-.4.27-.76.41-1.08.4-.36-.01-1.04-.2-1.55-.37-.63-.2-1.12-.31-1.08-.66.02-.18.27-.36.74-.55 2.92-1.27 4.86-2.11 5.83-2.51 2.78-1.16 3.35-1.36 3.73-1.36.08 0 .27.02.39.12.1.08.13.19.14.27-.01.06.01.24 0 .38z"/>
          </svg>
          <span>Войти через Telegram</span>
        </button>
      </div>

      <div v-else-if="state === 'loading'" class="content status-box">
        <div class="loader" />
        <span class="status-title">Отправка запроса...</span>
      </div>

      <div v-else-if="state === 'waiting'" class="content status-box">
        <div class="pulse-indicator">
          <span class="pulse-dot" />
        </div>
        <div class="status-texts">
          <h2 class="status-title">Подтвердите вход</h2>
          <p class="status-desc">Нажмите кнопку «Разрешить» в Telegram-боте</p>
        </div>
        <div class="countdown-badge">
          Осталось: <span class="num">{{ countdown }}</span> сек
        </div>
        <button class="btn-secondary" @click="cancelLogin">
          Отменить
        </button>
      </div>

      <div v-else-if="state === 'rejected'" class="content status-box">
        <div class="state-badge badge-danger">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <line x1="15" y1="9" x2="9" y2="15"/>
            <line x1="9" y1="9" x2="15" y2="15"/>
          </svg>
        </div>
        <h2 class="status-title">Вход отклонен</h2>
        <p class="status-desc">Запрос был отклонен пользователем в Telegram</p>
        <button class="btn-primary" @click="startLogin">
          Попробовать снова
        </button>
      </div>

      <div v-else-if="state === 'expired'" class="content status-box">
        <div class="state-badge badge-neutral">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <polyline points="12 6 12 12 16 14"/>
          </svg>
        </div>
        <h2 class="status-title">Время истекло</h2>
        <p class="status-desc">Срок действия запроса на авторизацию истек</p>
        <button class="btn-primary" @click="startLogin">
          Попробовать снова
        </button>
      </div>

      <div v-else-if="state === 'error'" class="content status-box">
        <div class="state-badge badge-warning">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10"/>
            <line x1="12" y1="8" x2="12" y2="12"/>
            <line x1="12" y1="16" x2="12.01" y2="16"/>
          </svg>
        </div>
        <h2 class="status-title">Не удалось войти</h2>
        <p class="status-desc">{{ errorMessage }}</p>
        <button class="btn-primary" @click="startLogin">
          Попробовать снова
        </button>
      </div>

      <div class="footer">
        <span>Личная сессия • 1 активное подключение</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { authApi } from '../api/client'
import { useAuthStore } from '../stores/auth'

type LoginState = 'idle' | 'loading' | 'waiting' | 'rejected' | 'expired' | 'error'

const router = useRouter()
const auth = useAuthStore()

const state = ref<LoginState>('idle')
const countdown = ref(120)
const errorMessage = ref('')

let pollTimer: ReturnType<typeof setInterval> | null = null
let countdownTimer: ReturnType<typeof setInterval> | null = null

function cleanupTimers() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

async function startLogin() {
  cleanupTimers()
  state.value = 'loading'
  errorMessage.value = ''

  try {
    const { data } = await authApi.requestLogin()
    state.value = 'waiting'
    countdown.value = data.expires_in

    countdownTimer = setInterval(() => {
      if (countdown.value > 0) {
        countdown.value--
      } else {
        cleanupTimers()
        state.value = 'expired'
      }
    }, 1000)

    pollTimer = setInterval(async () => {
      try {
        const checkRes = await authApi.checkLogin(data.login_id)
        if (checkRes.data.status === 'approved' && checkRes.data.access_token) {
          cleanupTimers()
          auth.setToken(checkRes.data.access_token)
          router.push('/stream')
        } else if (checkRes.data.status === 'rejected') {
          cleanupTimers()
          state.value = 'rejected'
        } else if (checkRes.data.status === 'expired') {
          cleanupTimers()
          state.value = 'expired'
        }
      } catch (err: any) {
        cleanupTimers()
        state.value = 'error'
        errorMessage.value = err.response?.data?.message || 'Ошибка соединения с сервером'
      }
    }, 1500)
  } catch (err: any) {
    state.value = 'error'
    const status = err.response?.status
    if (status === 409) {
      errorMessage.value = 'Запрос уже отправлен в Telegram. Подтвердите его или дождитесь завершения.'
    } else if (status === 429) {
      errorMessage.value = err.response?.data?.message || 'Слишком много попыток. Подождите перед повтором.'
    } else {
      errorMessage.value = err.response?.data?.message || 'Не удалось связаться с сервером.'
    }
  }
}

function cancelLogin() {
  cleanupTimers()
  state.value = 'idle'
}

onMounted(() => {
  if (auth.isAuth) {
    router.push('/stream')
  }
})

onUnmounted(() => {
  cleanupTimers()
})
</script>

<style scoped>
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background-color: #09090b;
  padding: 1.5rem;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  color: #f4f4f5;
}

.login-card {
  width: 100%;
  max-width: 380px;
  background-color: #121215;
  border: 1px solid #27272a;
  border-radius: 12px;
  padding: 2.25rem 2rem 1.5rem;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.4);
}

.header {
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  margin-bottom: 2rem;
}

.brand-icon {
  width: 42px;
  height: 42px;
  border-radius: 10px;
  background-color: #18181b;
  border: 1px solid #27272a;
  color: #fafafa;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 1rem;
}

.brand-name {
  font-size: 1.5rem;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: #fafafa;
  margin-bottom: 0.25rem;
}

.brand-desc {
  font-size: 0.85rem;
  color: #71717a;
}

.content {
  display: flex;
  flex-direction: column;
  align-items: stretch;
}

.status-box {
  align-items: center;
  text-align: center;
  gap: 0.85rem;
}

.status-title {
  font-size: 1.05rem;
  font-weight: 600;
  color: #fafafa;
}

.status-desc {
  font-size: 0.82rem;
  color: #a1a1aa;
  line-height: 1.45;
  margin-top: 0.2rem;
}

.btn-primary {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.6rem;
  width: 100%;
  padding: 0.75rem 1rem;
  border-radius: 8px;
  background-color: #fafafa;
  color: #09090b;
  font-size: 0.875rem;
  font-weight: 500;
  border: 1px solid #fafafa;
  cursor: pointer;
  transition: background-color 0.15s ease, opacity 0.15s ease;
}

.btn-primary:hover {
  background-color: #e4e4e7;
}

.btn-primary:active {
  opacity: 0.9;
}

.tg-icon {
  flex-shrink: 0;
}

.btn-secondary {
  width: 100%;
  padding: 0.65rem 1rem;
  border-radius: 8px;
  background-color: #18181b;
  color: #a1a1aa;
  font-size: 0.82rem;
  font-weight: 500;
  border: 1px solid #27272a;
  cursor: pointer;
  transition: background-color 0.15s ease, color 0.15s ease;
  margin-top: 0.5rem;
}

.btn-secondary:hover {
  background-color: #27272a;
  color: #fafafa;
}

.countdown-badge {
  display: inline-flex;
  align-items: center;
  padding: 0.25rem 0.75rem;
  border-radius: 6px;
  background-color: #18181b;
  border: 1px solid #27272a;
  color: #a1a1aa;
  font-size: 0.78rem;
  font-family: inherit;
}

.countdown-badge .num {
  font-variant-numeric: tabular-nums;
  font-weight: 600;
  color: #fafafa;
  margin: 0 0.2rem;
}

.state-badge {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 1px solid #27272a;
}

.badge-danger {
  background-color: rgba(239, 68, 68, 0.1);
  color: #ef4444;
  border-color: rgba(239, 68, 68, 0.2);
}

.badge-warning {
  background-color: rgba(245, 158, 11, 0.1);
  color: #f59e0b;
  border-color: rgba(245, 158, 11, 0.2);
}

.badge-neutral {
  background-color: #18181b;
  color: #a1a1aa;
}

.loader {
  width: 28px;
  height: 28px;
  border: 2px solid #27272a;
  border-top-color: #fafafa;
  border-radius: 50%;
  animation: spin 0.7s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.pulse-indicator {
  width: 14px;
  height: 14px;
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
}

.pulse-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background-color: #3b82f6;
  position: relative;
}

.pulse-dot::after {
  content: '';
  position: absolute;
  inset: -3px;
  border-radius: 50%;
  border: 1px solid #3b82f6;
  opacity: 0.6;
  animation: ping 1.5s cubic-bezier(0, 0, 0.2, 1) infinite;
}

@keyframes ping {
  75%, 100% {
    transform: scale(2.2);
    opacity: 0;
  }
}

.footer {
  margin-top: 2rem;
  padding-top: 1rem;
  border-top: 1px solid #1f1f23;
  text-align: center;
  font-size: 0.72rem;
  color: #52525b;
}
</style>
