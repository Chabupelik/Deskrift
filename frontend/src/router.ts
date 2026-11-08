import { createRouter, createWebHistory } from 'vue-router'
import LoginView from './views/LoginView.vue'
import StreamView from './views/StreamView.vue'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/', component: LoginView },
    { path: '/stream', component: StreamView },
  ],
})

export default router

