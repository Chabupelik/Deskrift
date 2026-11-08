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

    <div v-if="!isSupported" class="overlay">
      <div class="overlay-card">
        <div class="overlay-icon icon-danger">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10" />
            <line x1="12" y1="8" x2="12" y2="12" />
            <line x1="12" y1="16" x2="12.01" y2="16" />
          </svg>
        </div>
        <h2>WebCodecs не поддерживается</h2>
        <p>Для ультранизкой задержки требуется WebCodecs API (VideoDecoder).</p>
        <div class="hint-block">
          <div>• <b>Chrome / Edge / Brave:</b> поддерживается из коробки</div>
          <div>• <b>Firefox:</b> в <code>about:config</code> включите <code>dom.media.webcodecs.enabled</code></div>
        </div>
        <button class="btn-action" @click="returnToLogin">
          Вернуться назад
        </button>
      </div>
    </div>

    <div v-else-if="!stream.pcOnline && stream.connected" class="overlay">
      <div class="overlay-card">
        <div class="overlay-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <rect width="20" height="14" x="2" y="3" rx="2" />
            <line x1="8" x2="16" y1="21" y2="21" />
            <line x1="12" x2="12" y1="17" y2="21" />
          </svg>
        </div>
        <h2>Компьютер не в сети</h2>
        <p>Десктоп-агент Deskrift ожидает подключения на ПК</p>
      </div>
    </div>

    <div v-if="!stream.connected" class="overlay">
      <div class="overlay-card">
        <div class="overlay-icon icon-danger">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10" />
            <line x1="15" y1="9" x2="9" y2="15" />
            <line x1="9" y1="9" x2="15" y2="15" />
          </svg>
        </div>
        <h2>Сессия завершена</h2>
        <p>Соединение с сервером было закрыто</p>
        <button class="btn-action" @click="returnToLogin">
          Вернуться ко входу
        </button>
      </div>
    </div>

    <div class="toolbar">
      <span class="status-dot" :class="{ online: stream.pcOnline && stream.connected }" />
      <span class="status-label">{{ stream.pcOnline ? 'В сети' : 'Ожидание' }}</span>
      <button class="btn-exit" @click="returnToLogin" title="Завершить сессию">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <line x1="18" y1="6" x2="6" y2="18"/>
          <line x1="6" y1="6" x2="18" y2="18"/>
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
const isSupported = ref(typeof window !== 'undefined' && 'VideoDecoder' in window)

let ws: WebSocket | null = null
let decoder: any = null

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

  if (!isSupported.value) {
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
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
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
  background: rgba(9, 9, 11, 0.88);
  backdrop-filter: blur(10px);
  z-index: 20;
}

.overlay-card {
  background: #121215;
  border: 1px solid #27272a;
  border-radius: 12px;
  padding: 2.25rem 2rem;
  text-align: center;
  max-width: 360px;
  box-shadow: 0 10px 30px rgba(0, 0, 0, 0.6);
  display: flex;
  flex-direction: column;
  align-items: center;
}

.overlay-icon {
  width: 48px;
  height: 48px;
  border-radius: 10px;
  background-color: #18181b;
  border: 1px solid #27272a;
  color: #a1a1aa;
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 1.25rem;
}

.overlay-icon.icon-danger {
  color: #ef4444;
  background-color: rgba(239, 68, 68, 0.1);
  border-color: rgba(239, 68, 68, 0.2);
}

.overlay-card h2 {
  font-size: 1.25rem;
  font-weight: 600;
  color: #fafafa;
  margin-bottom: 0.35rem;
}

.overlay-card p {
  color: #71717a;
  font-size: 0.85rem;
  margin-bottom: 1.5rem;
  line-height: 1.4;
}

.hint-block {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
  background-color: #18181b;
  border: 1px solid #27272a;
  border-radius: 8px;
  padding: 0.75rem 1rem;
  font-size: 0.78rem;
  color: #a1a1aa;
  text-align: left;
  margin-bottom: 1.25rem;
  line-height: 1.4;
}

.hint-block code {
  color: #fafafa;
  background: #27272a;
  padding: 0.1rem 0.3rem;
  border-radius: 4px;
}

.btn-action {
  background: #fafafa;
  color: #09090b;
  border: 1px solid #fafafa;
  padding: 0.65rem 1.25rem;
  font-size: 0.85rem;
  font-weight: 500;
  border-radius: 8px;
  cursor: pointer;
  transition: background-color 0.15s ease;
}

.btn-action:hover {
  background: #e4e4e7;
}

.toolbar {
  position: absolute;
  top: 1rem;
  right: 1rem;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  background: rgba(18, 18, 21, 0.8);
  backdrop-filter: blur(8px);
  border: 1px solid #27272a;
  border-radius: 8px;
  padding: 0.35rem 0.65rem;
  z-index: 30;
  pointer-events: auto;
}

.status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: #71717a;
}

.status-dot.online {
  background: #22c55e;
}

.status-label {
  font-size: 0.75rem;
  color: #a1a1aa;
  font-weight: 500;
}

.btn-exit {
  background: transparent;
  border: none;
  color: #71717a;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  margin-left: 0.25rem;
  padding: 0.2rem;
  border-radius: 4px;
  transition: color 0.15s ease;
}

.btn-exit:hover {
  color: #fafafa;
}
</style>
