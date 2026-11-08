<template>
  <div class="login-wrapper">
    <div class="login-card">
      <div class="brand">
        <div class="logo-mark">⚡</div>
        <h1 class="logo-title">Deskrift</h1>
        <p class="logo-subtitle">Удаленный рабочий стол с низкой задержкой</p>
      </div>

      <div v-if="state === 'idle'" class="action-block">
        <button class="btn-primary" @click="startLogin">
          <span>Войти через Telegram</span>
        </button>
      </div>

      <div v-else-if="state === 'loading'" class="status-block">
        <div class="spinner" />
        <p class="status-text">Инициализация запроса...</p>
      </div>

      <div v-else-if="state === 'waiting'" class="status-block">
        <div class="pulse-ring" />
        <h3 class="status-title">Подтвердите вход</h3>
        <p class="status-desc">
          В Telegram отправлен запрос с кнопками подтверждения. Нажмите «Разрешить» в боте.
        </p>
        <div class="timer-badge">
          Осталось: {{ countdown }} сек
        </div>
        <button class="btn-secondary" @click="cancelLogin">
          Отмена
        </button>
      </div>

      <div v-else-if="state === 'rejected'" class="status-block">
        <div class="status-icon icon-error">✕</div>
        <h3 class="status-title">Вход отклонен</h3>
        <p class="status-desc">Запрос был отклонен в Telegram.</p>
        <button class="btn-primary" @click="startLogin">
          Попробовать снова
        </button>
      </div>

      <div v-else-if="state === 'expired'" class="status-block">
        <div class="status-icon icon-warning">⏱</div>
        <h3 class="status-title">Время истекло</h3>
        <p class="status-desc">Срок действия запроса на вход истек.</p>
        <button class="btn-primary" @click="startLogin">
          Попробовать снова
        </button>
      </div>

      <div v-else-if="state === 'error'" class="status-block">
        <div class="status-icon icon-error">!</div>
        <h3 class="status-title">Ошибка входа</h3>
        <p class="status-desc">{{ errorMessage }}</p>
        <button class="btn-primary" @click="startLogin">
          Попробовать снова
        </button>
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
        errorMessage.value = err.response?.data?.message || 'Ошибка проверки статуса'
      }
    }, 1500)
  } catch (err: any) {
    state.value = 'error'
    const status = err.response?.status
    if (status === 409) {
      errorMessage.value = 'Запрос уже ожидает подтверждения в Telegram. Откройте бота.'
    } else if (status === 429) {
      errorMessage.value = err.response?.data?.message || 'Слишком много попыток. Подождите перед повторной попыткой.'
    } else {
      errorMessage.value = err.response?.data?.message || 'Не удалось отправить запрос на вход.'
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
.login-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: radial-gradient(circle at 50% 20%, #1a1a24 0%, #0a0a0f 70%);
  padding: 1.5rem;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
}

.login-card {
  width: 100%;
  max-width: 420px;
  background: rgba(22, 22, 30, 0.75);
  backdrop-filter: blur(16px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 16px;
  padding: 2.5rem 2rem;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.4);
  text-align: center;
}

.brand {
  margin-bottom: 2rem;
}

.logo-mark {
  font-size: 2.5rem;
  margin-bottom: 0.5rem;
  filter: drop-shadow(0 0 12px rgba(99, 102, 241, 0.6));
}

.logo-title {
  font-size: 1.85rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: #fff;
  margin-bottom: 0.35rem;
}

.logo-subtitle {
  font-size: 0.9rem;
  color: #8e8ea0;
}

.action-block {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.btn-primary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  padding: 0.85rem 1.25rem;
  border-radius: 10px;
  background: #3b82f6;
  color: #fff;
  font-size: 0.95rem;
  font-weight: 600;
  border: none;
  cursor: pointer;
  transition: all 0.2s ease;
  box-shadow: 0 4px 14px rgba(59, 130, 246, 0.35);
}

.btn-primary:hover {
  background: #2563eb;
  transform: translateY(-1px);
  box-shadow: 0 6px 20px rgba(59, 130, 246, 0.45);
}

.btn-secondary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  padding: 0.75rem 1.25rem;
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.06);
  color: #c4c4d0;
  font-size: 0.9rem;
  font-weight: 500;
  border: 1px solid rgba(255, 255, 255, 0.08);
  cursor: pointer;
  transition: all 0.2s ease;
  margin-top: 1rem;
}

.btn-secondary:hover {
  background: rgba(255, 255, 255, 0.1);
  color: #fff;
}

.status-block {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.75rem;
}

.status-title {
  font-size: 1.2rem;
  font-weight: 600;
  color: #fff;
}

.status-desc {
  font-size: 0.88rem;
  color: #9ca3af;
  line-height: 1.4;
}

.timer-badge {
  display: inline-block;
  padding: 0.35rem 0.85rem;
  border-radius: 9999px;
  background: rgba(59, 130, 246, 0.15);
  border: 1px solid rgba(59, 130, 246, 0.3);
  color: #60a5fa;
  font-size: 0.82rem;
  font-weight: 600;
  margin-top: 0.25rem;
}

.status-icon {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 1.4rem;
  font-weight: bold;
}

.icon-error {
  background: rgba(239, 68, 68, 0.15);
  color: #ef4444;
  border: 1px solid rgba(239, 68, 68, 0.3);
}

.icon-warning {
  background: rgba(245, 158, 11, 0.15);
  color: #f59e0b;
  border: 1px solid rgba(245, 158, 11, 0.3);
}

.spinner {
  width: 36px;
  height: 36px;
  border: 3px solid rgba(255, 255, 255, 0.1);
  border-top-color: #3b82f6;
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
  margin-bottom: 0.5rem;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.pulse-ring {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  background: #3b82f6;
  animation: pulse 1.8s infinite ease-in-out;
  margin-bottom: 0.5rem;
}

@keyframes pulse {
  0% { transform: scale(0.9); box-shadow: 0 0 0 0 rgba(59, 130, 246, 0.7); }
  70% { transform: scale(1); box-shadow: 0 0 0 16px rgba(59, 130, 246, 0); }
  100% { transform: scale(0.9); box-shadow: 0 0 0 0 rgba(59, 130, 246, 0); }
}
</style>
