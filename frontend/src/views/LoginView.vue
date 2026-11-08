<template>
  <div class="login-page">
    <div class="login-card">
      <div class="header">
        <div class="brand-row">
          <svg class="brand-icon" width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
            <rect width="20" height="14" x="2" y="3" rx="2" />
            <line x1="8" x2="16" y1="21" y2="21" />
            <line x1="12" x2="12" y1="17" y2="21" />
          </svg>
          <span class="brand-name">Deskrift</span>
        </div>
        <p class="brand-tagline">Удаленный доступ сверхнизкой задержки</p>
      </div>

      <div class="content">
        <div v-if="state === 'idle'" class="state-pane">
          <button class="btn-primary" @click="startLogin">
            <svg class="tg-icon" width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm4.64 6.8c-.15 1.58-.8 5.42-1.13 7.19-.14.75-.42 1-.68 1.03-.58.05-1.02-.38-1.58-.75-.88-.58-1.38-.94-2.23-1.5-.99-.65-.35-1.01.22-1.59.15-.15 2.71-2.48 2.76-2.69a.2.2 0 00-.05-.18c-.06-.05-.14-.03-.21-.02-.09.02-1.49.95-4.22 2.79-.4.27-.76.41-1.08.4-.36-.01-1.04-.2-1.55-.37-.63-.2-1.12-.31-1.08-.66.02-.18.27-.36.74-.55 2.92-1.27 4.86-2.11 5.83-2.51 2.78-1.16 3.35-1.36 3.73-1.36.08 0 .27.02.39.12.1.08.13.19.14.27-.01.06.01.24 0 .38z"/>
            </svg>
            <span>Войти через Telegram</span>
          </button>
        </div>

        <div v-else-if="state === 'loading'" class="state-pane">
          <div class="loader-ring" />
          <p class="status-msg">Инициализация сессии...</p>
        </div>

        <div v-else-if="state === 'waiting'" class="state-pane">
          <div class="loader-ring" />
          <div class="waiting-details">
            <div class="status-title">Подтвердите вход</div>
            <p class="status-sub">Откройте Telegram-бота и нажмите «Разрешить»</p>
          </div>
          <div class="meta-row">
            <span class="mono-badge">{{ formattedTime }}</span>
          </div>
          <button class="btn-ghost" @click="cancelLogin">Отменить</button>
        </div>

        <div v-else-if="state === 'rejected'" class="state-pane">
          <div class="notice notice-danger">
            <span>Вход отклонен в Telegram</span>
          </div>
          <button class="btn-secondary" @click="startLogin">Попробовать снова</button>
        </div>

        <div v-else-if="state === 'expired'" class="state-pane">
          <div class="notice notice-warning">
            <span>Время ожидания подтверждения истекло</span>
          </div>
          <button class="btn-secondary" @click="startLogin">Попробовать снова</button>
        </div>

        <div v-else-if="state === 'error'" class="state-pane">
          <div class="notice notice-danger">
            <span>{{ errorMessage }}</span>
          </div>
          <button class="btn-secondary" @click="startLogin">Попробовать снова</button>
        </div>
      </div>

      <div class="footer">
        <span class="footer-meta">WebCodecs • NVENC 60 FPS • Direct Relay</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
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

const formattedTime = computed(() => {
  const mins = Math.floor(countdown.value / 60)
  const secs = countdown.value % 60
  return `0${mins}:${secs < 10 ? '0' : ''}${secs}`
})

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
        errorMessage.value = err.response?.data?.message || 'Сбой проверки статуса'
      }
    }, 1500)
  } catch (err: any) {
    state.value = 'error'
    const status = err.response?.status
    if (status === 409) {
      errorMessage.value = 'Запрос уже ожидает подтверждения в Telegram'
    } else if (status === 429) {
      errorMessage.value = err.response?.data?.message || 'Слишком много попыток. Подождите.'
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
.login-page {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background-color: #09090b;
  padding: 1rem;
}

.login-card {
  width: 100%;
  max-width: 360px;
  background: #0f0f12;
  border: 1px solid #1f1f24;
  border-radius: 12px;
  padding: 2rem 1.75rem 1.5rem;
  box-shadow: 0 4px 24px rgba(0, 0, 0, 0.4);
}

.header {
  margin-bottom: 1.75rem;
}

.brand-row {
  display: flex;
  align-items: center;
  gap: 0.6rem;
  margin-bottom: 0.35rem;
}

.brand-icon {
  color: #f4f4f5;
}

.brand-name {
  font-size: 1.15rem;
  font-weight: 600;
  letter-spacing: -0.02em;
  color: #f4f4f5;
}

.brand-tagline {
  font-size: 0.8125rem;
  color: #71717a;
  line-height: 1.35;
}

.content {
  min-height: 120px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.state-pane {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 1rem;
  width: 100%;
}

.btn-primary {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  width: 100%;
  height: 40px;
  padding: 0 1rem;
  background: #f4f4f5;
  color: #09090b;
  border: 1px solid #e4e4e7;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.15s ease, transform 0.05s ease;
}

.btn-primary:hover {
  background: #e4e4e7;
}

.btn-primary:active {
  transform: scale(0.99);
}

.btn-secondary {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 38px;
  background: #18181b;
  color: #f4f4f5;
  border: 1px solid #27272a;
  border-radius: 8px;
  font-size: 0.875rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.btn-secondary:hover {
  background: #27272a;
}

.btn-ghost {
  background: transparent;
  border: none;
  color: #71717a;
  font-size: 0.8125rem;
  cursor: pointer;
  padding: 0.25rem 0.5rem;
  border-radius: 4px;
  transition: color 0.15s ease;
}

.btn-ghost:hover {
  color: #d4d4d8;
}

.tg-icon {
  flex-shrink: 0;
}

.loader-ring {
  width: 24px;
  height: 24px;
  border: 2px solid #27272a;
  border-top-color: #f4f4f5;
  border-radius: 50%;
  animation: rotate 0.75s linear infinite;
}

@keyframes rotate {
  to { transform: rotate(360deg); }
}

.status-msg {
  font-size: 0.8125rem;
  color: #a1a1aa;
}

.waiting-details {
  text-align: center;
}

.status-title {
  font-size: 0.9375rem;
  font-weight: 500;
  color: #f4f4f5;
  margin-bottom: 0.25rem;
}

.status-sub {
  font-size: 0.8125rem;
  color: #71717a;
  line-height: 1.4;
}

.meta-row {
  display: flex;
  align-items: center;
  justify-content: center;
}

.mono-badge {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.75rem;
  color: #a1a1aa;
  background: #18181b;
  border: 1px solid #27272a;
  padding: 0.2rem 0.6rem;
  border-radius: 6px;
  letter-spacing: 0.04em;
}

.notice {
  width: 100%;
  padding: 0.65rem 0.85rem;
  border-radius: 8px;
  font-size: 0.8125rem;
  line-height: 1.4;
  text-align: center;
}

.notice-danger {
  background: #191012;
  border: 1px solid #381a1f;
  color: #f87171;
}

.notice-warning {
  background: #18140c;
  border: 1px solid #382d16;
  color: #fbbf24;
}

.footer {
  margin-top: 1.75rem;
  padding-top: 1rem;
  border-top: 1px solid #1a1a1f;
  text-align: center;
}

.footer-meta {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.6875rem;
  color: #52525b;
  letter-spacing: 0.02em;
}
</style>
