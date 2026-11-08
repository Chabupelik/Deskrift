<template>
  <div class="stream-container" ref="containerRef" tabindex="0" @keydown="onKeyDown" @keyup="onKeyUp" @paste="onPaste">
    <canvas
      ref="canvasRef"
      class="stream-canvas"
      @mousemove="onMouseMove"
      @mousedown="onMouseDown"
      @mouseup="onMouseUp"
      @contextmenu.prevent
    />

    <div v-if="!stream.pcOnline && stream.connected" class="overlay">
      <div class="overlay-card">
        <div class="overlay-icon">🖥️</div>
        <h2>Компьютер не в сети</h2>
        <p>Десктоп-агент Deskrift ожидает подключения на ПК</p>
      </div>
    </div>

    <div v-if="!stream.connected" class="overlay">
      <div class="overlay-card">
        <div class="overlay-icon">🛑</div>
        <h2>Сессия завершена</h2>
        <p>Соединение с сервером закрыто</p>
        <button class="btn-reconnect" @click="returnToLogin">
          Вернуться ко входу
        </button>
      </div>
    </div>

    <div class="toolbar">
      <span class="status-dot" :class="{ online: stream.pcOnline && stream.connected }" />
      <span class="status-label">{{ stream.pcOnline ? 'Онлайн' : 'Ожидание ПК' }}</span>
      <button class="btn-exit" @click="returnToLogin" title="Завершить сессию">✕</button>
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

let ws: WebSocket | null = null
let decoder: VideoDecoder | null = null

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
      console.error('WebCodecs VideoDecoder error:', e)
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
        console.error('Failed to parse control message:', err)
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
.stream-container {
  position: relative;
  width: 100vw;
  height: 100vh;
  overflow: hidden;
  background: #000;
  outline: none;
}

.stream-canvas {
  display: block;
  width: 100%;
  height: 100%;
  cursor: crosshair;
}

.overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(10, 10, 15, 0.85);
  backdrop-filter: blur(12px);
  z-index: 20;
}

.overlay-card {
  background: rgba(26, 26, 36, 0.9);
  border: 1px solid rgba(255, 255, 255, 0.1);
  border-radius: 16px;
  padding: 2.5rem;
  text-align: center;
  max-width: 400px;
  box-shadow: 0 20px 40px rgba(0, 0, 0, 0.5);
}

.overlay-icon {
  font-size: 3rem;
  margin-bottom: 1rem;
}

.overlay-card h2 {
  font-size: 1.4rem;
  font-weight: 700;
  color: #fff;
  margin-bottom: 0.5rem;
}

.overlay-card p {
  color: #9ca3af;
  font-size: 0.92rem;
  margin-bottom: 1.5rem;
  line-height: 1.4;
}

.btn-reconnect {
  background: #3b82f6;
  color: #fff;
  border: none;
  padding: 0.75rem 1.5rem;
  font-size: 0.95rem;
  font-weight: 600;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s;
}

.btn-reconnect:hover {
  background: #2563eb;
}

.toolbar {
  position: absolute;
  top: 1rem;
  right: 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: rgba(20, 20, 28, 0.7);
  backdrop-filter: blur(10px);
  border: 1px solid rgba(255, 255, 255, 0.08);
  border-radius: 9999px;
  padding: 0.4rem 0.85rem;
  z-index: 30;
  pointer-events: auto;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #ef4444;
}

.status-dot.online {
  background: #22c55e;
  box-shadow: 0 0 8px #22c55e;
}

.status-label {
  font-size: 0.78rem;
  color: #e2e8f0;
  font-weight: 500;
}

.btn-exit {
  background: transparent;
  border: none;
  color: #94a3b8;
  font-size: 0.9rem;
  cursor: pointer;
  margin-left: 0.25rem;
  padding: 0.1rem 0.3rem;
  border-radius: 4px;
  transition: all 0.2s;
}

.btn-exit:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.1);
}
</style>
