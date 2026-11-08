<template>
  <div
    class="stream-viewport"
    ref="containerRef"
    tabindex="0"
    @keydown="onKeyDown"
    @keyup="onKeyUp"
    @paste="onPaste"
    @mousemove="handleActivity"
  >
    <canvas
      ref="canvasRef"
      class="video-canvas"
      @mousemove="onMouseMove"
      @mousedown="onMouseDown"
      @mouseup="onMouseUp"
      @contextmenu.prevent
    />

    <div v-if="!stream.pcOnline && stream.connected" class="status-overlay">
      <div class="overlay-modal">
        <svg class="modal-icon" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
          <rect width="20" height="14" x="2" y="3" rx="2" />
          <line x1="8" x2="16" y1="21" y2="21" />
          <line x1="12" x2="12" y1="17" y2="21" />
        </svg>
        <div class="modal-title">Компьютер не в сети</div>
        <p class="modal-desc">Десктоп-агент Deskrift ожидает подключения на ПК</p>
      </div>
    </div>

    <div v-if="!stream.connected" class="status-overlay">
      <div class="overlay-modal">
        <svg class="modal-icon" width="28" height="28" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75">
          <circle cx="12" cy="12" r="10" />
          <line x1="15" x2="9" y1="9" y2="15" />
          <line x1="9" x2="15" y1="9" y2="15" />
        </svg>
        <div class="modal-title">Сессия завершена</div>
        <p class="modal-desc">Соединение с сервером закрыто</p>
        <button class="btn-action" @click="returnToLogin">
          Вернуться ко входу
        </button>
      </div>
    </div>

    <div class="hud" :class="{ 'hud-dimmed': isIdle }">
      <div class="hud-indicator" :class="{ active: stream.pcOnline && stream.connected }" />
      <span class="hud-label">{{ stream.pcOnline ? 'LIVE' : 'STANDBY' }}</span>
      <span class="hud-res">{{ stream.remoteWidth }}×{{ stream.remoteHeight }}</span>
      <button class="hud-close" @click="returnToLogin" title="Завершить сессию">
        <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
          <line x1="18" x2="6" y1="6" y2="18" />
          <line x1="6" x2="18" y1="6" y2="18" />
        </svg>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAuthStore } from '../stores/auth'
import { useStreamStore } from '../stores/stream'

const router = useRouter()
const auth = useAuthStore()
const stream = useStreamStore()

const containerRef = ref<HTMLDivElement>()
const canvasRef = ref<HTMLCanvasElement>()
const isIdle = ref(false)

let ws: WebSocket | null = null
let decoder: VideoDecoder | null = null
let idleTimer: ReturnType<typeof setTimeout> | null = null

function handleActivity() {
  isIdle.value = false
  if (idleTimer) clearTimeout(idleTimer)
  idleTimer = setTimeout(() => {
    isIdle.value = true
  }, 2500)
}

function updateCanvasSize() {
  const canvas = canvasRef.value
  if (!canvas) return
  canvas.width = window.innerWidth
  canvas.height = window.innerHeight
}

function sendMouseEvent(type: 'mousemove' | 'mousedown' | 'mouseup', e: MouseEvent, button?: 'left' | 'right') {
  if (!ws || ws.readyState !== WebSocket.OPEN || !stream.pcOnline) return
  const canvas = canvasRef.value
  if (!canvas) return

  const rect = canvas.getBoundingClientRect()
  const xRatio = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width))
  const yRatio = Math.max(0, Math.min(1, (e.clientY - rect.top) / rect.height))
  const x = Math.round(xRatio * stream.remoteWidth)
  const y = Math.round(yRatio * stream.remoteHeight)

  const payload: Record<string, any> = { type, x, y }
  if (button) {
    payload.button = button
  }
  ws.send(JSON.stringify(payload))
}

function onMouseMove(e: MouseEvent) {
  sendMouseEvent('mousemove', e)
}

function onMouseDown(e: MouseEvent) {
  e.preventDefault()
  containerRef.value?.focus()
  const button = e.button === 2 ? 'right' : 'left'
  sendMouseEvent('mousedown', e, button)
}

function onMouseUp(e: MouseEvent) {
  e.preventDefault()
  const button = e.button === 2 ? 'right' : 'left'
  sendMouseEvent('mouseup', e, button)
}

function onKeyDown(e: KeyboardEvent) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return
  if (e.key === 'Tab' || e.key === 'Alt' || e.key.startsWith('F')) {
    e.preventDefault()
  }
  ws.send(JSON.stringify({ type: 'keydown', key: e.keyCode }))
}

function onKeyUp(e: KeyboardEvent) {
  if (!ws || ws.readyState !== WebSocket.OPEN) return
  ws.send(JSON.stringify({ type: 'keyup', key: e.keyCode }))
}

function onPaste(e: ClipboardEvent) {
  const text = e.clipboardData?.getData('text')
  if (text && ws && ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify({ type: 'clipboard', text }))
  }
}

function returnToLogin() {
  auth.logout()
  stream.reset()
  router.push('/')
}

onMounted(() => {
  if (!auth.token) {
    router.push('/')
    return
  }

  const canvas = canvasRef.value!
  const ctx = canvas.getContext('2d', { alpha: false })!
  updateCanvasSize()
  window.addEventListener('resize', updateCanvasSize)

  decoder = new VideoDecoder({
    output(frame) {
      if (frame.displayWidth && frame.displayHeight) {
        stream.remoteWidth = frame.displayWidth
        stream.remoteHeight = frame.displayHeight
      }
      ctx.drawImage(frame, 0, 0, canvas.width, canvas.height)
      frame.close()
    },
    error(e) {
      console.error('VideoDecoder error:', e)
    },
  })

  decoder.configure({
    codec: 'avc1.42001f',
    hardwareAcceleration: 'prefer-hardware',
    optimizeForLatency: true,
  })

  const wsProtocol = location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${wsProtocol}//${location.host}/ws/browser?token=${auth.token}`

  ws = new WebSocket(wsUrl)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => {
    stream.connected = true
    containerRef.value?.focus()
    handleActivity()
  }

  ws.onclose = () => {
    stream.connected = false
    stream.pcOnline = false
  }

  ws.onmessage = (event) => {
    if (typeof event.data === 'string') {
      try {
        const msg = JSON.parse(event.data)
        if (msg.type === 'pc_status') {
          stream.pcOnline = !!msg.online
        }
      } catch (err) {
        console.error('Parse error:', err)
      }
      return
    }

    if (event.data instanceof ArrayBuffer && decoder && decoder.state === 'configured') {
      try {
        const chunk = new EncodedVideoChunk({
          type: 'key',
          timestamp: performance.now() * 1000,
          data: event.data,
        })
        decoder.decode(chunk)
      } catch (err) {
        console.error('Decode error:', err)
      }
    }
  }
})

onUnmounted(() => {
  window.removeEventListener('resize', updateCanvasSize)
  if (idleTimer) clearTimeout(idleTimer)
  if (ws) {
    ws.close()
    ws = null
  }
  if (decoder) {
    try {
      decoder.close()
    } catch {
      // Ignored on teardown
    }
    decoder = null
  }
  stream.reset()
})
</script>

<style scoped>
.stream-viewport {
  position: relative;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: #000;
  outline: none;
}

.video-canvas {
  display: block;
  width: 100%;
  height: 100%;
  cursor: crosshair;
}

.status-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(9, 9, 11, 0.92);
  z-index: 10;
}

.overlay-modal {
  background: #111115;
  border: 1px solid #1f1f26;
  border-radius: 12px;
  padding: 2rem 2.25rem;
  text-align: center;
  max-width: 360px;
  width: 90%;
  display: flex;
  flex-direction: column;
  align-items: center;
  box-shadow: 0 4px 30px rgba(0, 0, 0, 0.5);
}

.modal-icon {
  color: #71717a;
  margin-bottom: 1rem;
}

.modal-title {
  font-size: 1rem;
  font-weight: 600;
  color: #f4f4f5;
  margin-bottom: 0.35rem;
}

.modal-desc {
  font-size: 0.8125rem;
  color: #71717a;
  line-height: 1.4;
  margin-bottom: 1.25rem;
}

.btn-action {
  height: 36px;
  padding: 0 1.25rem;
  background: #f4f4f5;
  color: #09090b;
  border: 1px solid #e4e4e7;
  border-radius: 7px;
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.btn-action:hover {
  background: #e4e4e7;
}

.hud {
  position: absolute;
  top: 0.75rem;
  right: 0.75rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: rgba(15, 15, 18, 0.85);
  backdrop-filter: blur(8px);
  border: 1px solid #1f1f24;
  border-radius: 6px;
  padding: 0.3rem 0.6rem;
  z-index: 20;
  transition: opacity 0.3s ease;
  user-select: none;
}

.hud-dimmed {
  opacity: 0.15;
}

.hud-dimmed:hover {
  opacity: 1;
}

.hud-indicator {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: #71717a;
}

.hud-indicator.active {
  background: #22c55e;
}

.hud-label {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.6875rem;
  font-weight: 600;
  color: #d4d4d8;
  letter-spacing: 0.05em;
}

.hud-res {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
  font-size: 0.6875rem;
  color: #71717a;
  border-left: 1px solid #27272a;
  padding-left: 0.5rem;
}

.hud-close {
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  color: #71717a;
  cursor: pointer;
  padding: 2px;
  margin-left: 0.25rem;
  border-radius: 3px;
  transition: color 0.15s ease, background-color 0.15s ease;
}

.hud-close:hover {
  color: #f4f4f5;
  background: rgba(255, 255, 255, 0.08);
}
</style>
