<template>
  <div class="login">
    <h1>Deskrift</h1>
    <p>Войди через Telegram бота чтобы получить доступ</p>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'

const auth = useAuthStore()
const router = useRouter()

onMounted(() => {
  // Если пришли из Telegram WebApp — initData уже есть
  const tg = (window as any).Telegram?.WebApp
  if (tg?.initData) {
    auth.loginWithTelegram(tg.initData).then(() => {
      router.push('/stream')
    })
  } else if (auth.isAuth) {
    router.push('/stream')
  }
})
</script>

<style scoped>
.login {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100vh;
  gap: 1rem;
}
</style>

