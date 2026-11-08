<template>
  <canvas ref="canvasRef" class="stream-canvas" />
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, ref } from 'vue'
import { useAuthStore } from '../stores/auth'
import { useStreamStore } from '../stores/stream'

const canvasRef = ref<HTMLCanvasElement>()
const auth = useAuthStore()
const stream = useStreamStore()

let ws: WebSocket | null = null
let decoder: VideoDecoder | null = null

onMounted(() => {
  const canvas = canvasRef.value!
  const ctx = canvas.getContext('2d')!

  canvas.width = window.innerWidth
  canvas.height = window.innerHeight

  decoder = new VideoDecoder({
    output(frame) {
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
  })

  const wsUrl = `${location.protocol === 'https:' ? 'wss' : 'ws'}://${location.host}/ws/browser?token=${auth.token}`
  ws = new WebSocket(wsUrl)
  ws.binaryType = 'arraybuffer'

  ws.onopen = () => { stream.connected = true }
  ws.onclose = () => { stream.connected = false }

  ws.onmessage = (event) => {
    const chunk = new EncodedVideoChunk({
      type: 'key',
      timestamp: performance.now() * 1000,
      data: event.data,
    })
    decoder?.decode(chunk)
  }
})

onUnmounted(() => {
  ws?.close()
  decoder?.close()
})
</script>

<style scoped>
.stream-canvas {
  display: block;
  width: 100%;
  height: 100%;
}
</style>
