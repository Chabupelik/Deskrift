<template>
  <div class="input-capture" @mousemove="onMouseMove" @mousedown="onMouseDown" @mouseup="onMouseUp" @keydown="onKeyDown" @keyup="onKeyUp" tabindex="0" />
</template>

<script setup lang="ts">
import { inject } from 'vue'

// ws инжектируется из StreamView (TODO: через store)
const ws = inject<WebSocket>('ws')

function send(data: object) {
  ws?.send(JSON.stringify(data))
}

function onMouseMove(e: MouseEvent) {
  send({ type: 'mousemove', x: e.clientX, y: e.clientY })
}

function onMouseDown(e: MouseEvent) {
  send({ type: 'mousedown', button: e.button === 0 ? 'left' : 'right' })
}

function onMouseUp(e: MouseEvent) {
  send({ type: 'mouseup', button: e.button === 0 ? 'left' : 'right' })
}

function onKeyDown(e: KeyboardEvent) {
  e.preventDefault()
  send({ type: 'keydown', key: e.keyCode })
}

function onKeyUp(e: KeyboardEvent) {
  send({ type: 'keyup', key: e.keyCode })
}
</script>

<style scoped>
.input-capture {
  position: absolute;
  inset: 0;
  outline: none;
  cursor: none;
}
</style>

