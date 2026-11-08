<template>
  <div class="stream-container" ref="containerRef" tabindex="0" @keydown="onKeyDown" @keyup="onKeyUp" @paste="onPaste">
    <div class="canvas-wrapper">
      <canvas
        ref="canvasRef"
        class="stream-canvas"
        width="1920"
        height="1080"
        @mousemove="onMouseMove"
        @mousedown="onMouseDown"
        @mouseup="onMouseUp"
        @wheel.prevent="onWheel"
        @contextmenu.prevent
      />
    </div>

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
          <div>• <b>Firefox / LibreWolf:</b> в <code>about:config</code>:</div>
          <div style="padding-left: 1rem; margin-top: 0.25rem;">1. <code>dom.media.webcodecs.enabled</code> = <code>true</code></div>
          <div style="padding-left: 1rem;">2. <code>privacy.resistFingerprinting</code> = <code>false</code> (или исключите домен)</div>
          <div style="padding-left: 1rem; margin-top: 0.25rem; font-style: italic;">После изменения перезапустите браузер</div>
        </div>
        <button class="btn-action" @click="returnToLogin">
          Вернуться назад
        </button>
      </div>
    </div>

    <div v-else-if="isDisconnected" class="overlay">
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

    <div v-else-if="!stream.connected" class="overlay">
      <div class="overlay-card">
        <div class="overlay-icon">
          <svg width="32" height="32" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75" stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10" />
            <polyline points="12 6 12 12 14 14" />
          </svg>
        </div>
        <h2>Подключение</h2>
        <p>Установка связи с сервером...</p>
      </div>
    </div>

    <div v-else-if="!stream.pcOnline" class="overlay">
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
const isDisconnected = ref(false)

let ws: WebSocket | null = null
let decoder: any = null
let lastSentX = -1
let lastSentY = -1
let mouseRafId = 0
let pendingMouseMove: { x: number; y: number } | null = null

async function getSupportedH264Codec(): Promise<string> {
  const candidates = [
    'avc1.42002a',
    'avc1.42e02a',
    'avc1.42001f',
    'avc1.42e01f',
  ]
  if (typeof VideoDecoder !== 'undefined' && 'isConfigSupported' in VideoDecoder) {
    for (const codec of candidates) {
      try {
        const res = await (VideoDecoder as any).isConfigSupported({
          codec,
          hardwareAcceleration: 'prefer-hardware',
          optimizeForLatency: true,
        })
        if (res && res.supported) {
          return codec
        }
      } catch {}
    }
  }
  return 'avc1.42002a'
}

function sendMouseEvent(type: 'mousemove' | 'mousedown' | 'mouseup', e: MouseEvent, button?: 'left' | 'right' | 'middle') {
  if (!ws || ws.readyState !== WebSocket.OPEN || !stream.pcOnline) return
  const canvas = canvasRef.value
  if (!canvas) return

  const rect = canvas.getBoundingClientRect()
  if (rect.width === 0 || rect.height === 0) return

  const xRatio = Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width))
  const yRatio = Math.max(0, Math.min(1, (e.clientY - rect.top) / rect.height))
  const x = Math.round(xRatio * 1920)
  const y = Math.round(yRatio * 1080)

  if (type === 'mousemove') {
    if (x === lastSentX && y === lastSentY) return
    pendingMouseMove = { x, y }
    if (!mouseRafId) {
      mouseRafId = requestAnimationFrame(() => {
        mouseRafId = 0
        if (pendingMouseMove && ws && ws.readyState === WebSocket.OPEN) {
          lastSentX = pendingMouseMove.x
          lastSentY = pendingMouseMove.y
          ws.send(JSON.stringify({ type: 'mousemove', x: lastSentX, y: lastSentY }))
          pendingMouseMove = null
        }
      })
    }
    return
  }

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
  let button: 'left' | 'right' | 'middle' = 'left'
  if (e.button === 2) button = 'right'
  else if (e.button === 1) button = 'middle'
  sendMouseEvent('mousedown', e, button)
}

function onMouseUp(e: MouseEvent) {
  e.preventDefault()
  let button: 'left' | 'right' | 'middle' = 'left'
  if (e.button === 2) button = 'right'
  else if (e.button === 1) button = 'middle'
  sendMouseEvent('mouseup', e, button)
}

function onWheel(e: WheelEvent) {
  if (!ws || ws.readyState !== WebSocket.OPEN || !stream.pcOnline) return
  const delta = e.deltaY < 0 ? 120 : -120
  ws.send(JSON.stringify({ type: 'wheel', delta }))
}

function onKeyDown(e: KeyboardEvent) {
  if (!ws || ws.readyState !== WebSocket.OPEN || !stream.pcOnline) return
  if (e.key === 'Tab' || e.key === 'Alt' || e.key.startsWith('F')) {
    e.preventDefault()
  }
  ws.send(JSON.stringify({ type: 'keydown', key: e.keyCode }))
}

function onKeyUp(e: KeyboardEvent) {
  if (!ws || ws.readyState !== WebSocket.OPEN || !stream.pcOnline) return
  ws.send(JSON.stringify({ type: 'keyup', key: e.keyCode }))
}

function onPaste(e: ClipboardEvent) {
  if (!ws || ws.readyState !== WebSocket.OPEN || !stream.pcOnline) return
  const text = e.clipboardData?.getData('text')
  if (text) {
    ws.send(JSON.stringify({ type: 'clipboard', text }))
  }
}

function returnToLogin() {
  isDisconnected.value = false
  auth.logout()
  stream.reset()
  router.push('/')
}

onMounted(async () => {
  if (!auth.token) {
    router.push('/')
    return
  }

  if (!isSupported.value) {
    return
  }

  const canvas = canvasRef.value!
  const ctx = canvas.getContext('2d', { alpha: false })!

  let hasKeyframe = false

  decoder = new VideoDecoder({
    output(frame) {
      ctx.drawImage(frame, 0, 0, 1920, 1080)
      frame.close()
    },
    error(e) {
      console.error('WebCodecs VideoDecoder error:', e)
    },
  })

  const codec = await getSupportedH264Codec()
  decoder.configure({
    codec,
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
    isDisconnected.value = true
    hasKeyframe = false
  }

  ws.onerror = () => {
    stream.connected = false
    stream.pcOnline = false
    isDisconnected.value = true
    hasKeyframe = false
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
        const u8 = new Uint8Array(event.data)
        if (u8.length < 2) return

        const isKey = u8[0] === 1
        if (!hasKeyframe) {
          if (!isKey) return
          hasKeyframe = true
        }

        if (decoder.decodeQueueSize > 2 && !isKey) {
          return
        }

        const chunkData = event.data.slice(1)
        const chunk = new EncodedVideoChunk({
          type: isKey ? 'key' : 'delta',
          timestamp: performance.now() * 1000,
          data: chunkData,
        })
        decoder.decode(chunk)
      } catch (err) {
        console.error('Decode error:', err)
      }
    }
  }
})

onUnmounted(() => {
  if (mouseRafId) {
    cancelAnimationFrame(mouseRafId)
    mouseRafId = 0
  }
  if (ws) {
    ws.close()
    ws = null
  }
  if (decoder) {
    try {
      decoder.close()
    } catch {}
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
  display: flex;
  align-items: center;
  justify-content: center;
  outline: none;
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
}

.canvas-wrapper {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100vw;
  height: 100vh;
}

.stream-canvas {
  max-width: 100vw;
  max-height: 100vh;
  aspect-ratio: 16 / 9;
  object-fit: contain;
  cursor: default;
  display: block;
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
